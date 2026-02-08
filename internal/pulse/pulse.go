package pulse

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/HerbHall/subnetree/pkg/analytics"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/roles"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin            = (*Module)(nil)
	_ plugin.HTTPProvider      = (*Module)(nil)
	_ plugin.HealthChecker     = (*Module)(nil)
	_ plugin.EventSubscriber   = (*Module)(nil)
	_ roles.MonitoringProvider = (*Module)(nil)
)

// Module implements the Pulse monitoring plugin.
type Module struct {
	logger    *zap.Logger
	cfg       PulseConfig
	store     *PulseStore
	bus       plugin.EventBus
	plugins   plugin.PluginResolver
	scheduler *Scheduler
	checker   Checker
	alerter   *Alerter

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new Pulse plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "pulse",
		Version:     "0.1.0",
		Description: "Device monitoring and alerting",
		Roles:       []string{roles.RoleMonitoring},
		Required:    false,
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger

	m.cfg = DefaultConfig()
	if deps.Config != nil {
		if err := deps.Config.Unmarshal(&m.cfg); err != nil {
			return fmt.Errorf("unmarshal pulse config: %w", err)
		}
	}

	if deps.Store != nil {
		if err := deps.Store.Migrate(context.Background(), "pulse", migrations()); err != nil {
			return fmt.Errorf("pulse migrations: %w", err)
		}
		m.store = NewPulseStore(deps.Store.DB())
	}

	m.bus = deps.Bus
	m.plugins = deps.Plugins

	m.logger.Info("pulse module initialized",
		zap.Duration("check_interval", m.cfg.CheckInterval),
		zap.Duration("ping_timeout", m.cfg.PingTimeout),
		zap.Int("ping_count", m.cfg.PingCount),
		zap.Int("max_workers", m.cfg.MaxWorkers),
	)
	return nil
}

func (m *Module) Start(_ context.Context) error {
	m.ctx, m.cancel = context.WithCancel(context.Background())

	m.checker = NewICMPChecker(m.cfg.PingTimeout, m.cfg.PingCount)

	if m.store != nil {
		m.alerter = NewAlerter(m.store, m.bus, m.cfg.ConsecutiveFailures, m.logger)

		m.scheduler = NewScheduler(
			m.store,
			m.executeCheck,
			m.cfg.CheckInterval,
			m.cfg.MaxWorkers,
			m.logger,
		)
		m.scheduler.Start(m.ctx)
	}

	m.startMaintenance()

	m.logger.Info("pulse module started")
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	if m.scheduler != nil {
		m.scheduler.Stop()
	}
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	m.logger.Info("pulse module stopped")
	return nil
}

// executeCheck runs an ICMP check, stores the result, processes alerts, and publishes metrics.
func (m *Module) executeCheck(ctx context.Context, check Check) {
	result, err := m.checker.Check(ctx, check.Target)
	if err != nil {
		m.logger.Debug("check returned error",
			zap.String("check_id", check.ID),
			zap.String("target", check.Target),
			zap.Error(err),
		)
	}
	if result == nil {
		return
	}

	result.CheckID = check.ID
	result.DeviceID = check.DeviceID

	// Store the result.
	if err := m.store.InsertResult(ctx, result); err != nil {
		m.logger.Warn("failed to store check result",
			zap.String("check_id", check.ID),
			zap.Error(err),
		)
	}

	// Process alert state machine.
	if m.alerter != nil {
		m.alerter.ProcessResult(ctx, check, result)
	}

	// Publish metrics to the event bus for Insight consumption.
	m.publishMetrics(ctx, check, result)
}

// publishMetrics emits metric points for analytics processing.
func (m *Module) publishMetrics(ctx context.Context, check Check, result *CheckResult) {
	if m.bus == nil {
		return
	}

	now := time.Now().UTC()
	successVal := 0.0
	if result.Success {
		successVal = 1.0
	}

	metrics := []analytics.MetricPoint{
		{DeviceID: check.DeviceID, MetricName: "ping_latency_ms", Value: result.LatencyMs, Timestamp: now},
		{DeviceID: check.DeviceID, MetricName: "ping_packet_loss", Value: result.PacketLoss, Timestamp: now},
		{DeviceID: check.DeviceID, MetricName: "ping_success", Value: successVal, Timestamp: now},
	}

	m.bus.PublishAsync(ctx, plugin.Event{
		Topic:     TopicMetricsCollected,
		Source:    "pulse",
		Timestamp: now,
		Payload:   metrics,
	})
}

// -- plugin.HealthChecker --

// Health implements plugin.HealthChecker.
func (m *Module) Health(_ context.Context) plugin.HealthStatus {
	details := map[string]string{}

	if m.scheduler != nil && m.scheduler.Running() {
		details["scheduler"] = "running"
	} else {
		details["scheduler"] = "stopped"
	}

	if m.store != nil {
		details["store"] = "connected"
	} else {
		details["store"] = "unavailable"
	}

	status := "healthy"
	if m.store == nil {
		status = "degraded"
	}

	return plugin.HealthStatus{
		Status:  status,
		Details: details,
	}
}

// -- plugin.EventSubscriber --

// Subscriptions implements plugin.EventSubscriber.
func (m *Module) Subscriptions() []plugin.Subscription {
	return []plugin.Subscription{
		{Topic: TopicDeviceDiscovered, Handler: m.handleDeviceDiscovered},
	}
}

// -- roles.MonitoringProvider --

// Status implements roles.MonitoringProvider.
func (m *Module) Status(ctx context.Context, deviceID string) (*roles.MonitorStatus, error) {
	if m.store == nil {
		return nil, fmt.Errorf("pulse store not available")
	}

	// Get latest result for this device.
	results, err := m.store.ListResults(ctx, deviceID, 1)
	if err != nil {
		return nil, fmt.Errorf("query results: %w", err)
	}

	status := &roles.MonitorStatus{
		DeviceID: deviceID,
		Healthy:  true,
		Message:  "no check data available",
	}

	if len(results) > 0 {
		latest := results[0]
		status.Healthy = latest.Success
		status.CheckedAt = latest.CheckedAt
		if latest.Success {
			status.Message = fmt.Sprintf("ping OK (%.1fms, %.0f%% loss)", latest.LatencyMs, latest.PacketLoss*100)
		} else {
			status.Message = "ping failed"
			if latest.ErrorMessage != "" {
				status.Message = latest.ErrorMessage
			}
		}
	}

	// Check for active alerts.
	alerts, err := m.store.ListActiveAlerts(ctx, deviceID)
	if err == nil && len(alerts) > 0 {
		status.Healthy = false
		status.Message = alerts[0].Message
	}

	return status, nil
}

// Routes is implemented in handlers.go.
