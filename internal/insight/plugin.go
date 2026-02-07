package insight

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/HerbHall/subnetree/pkg/analytics"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/roles"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin          = (*Module)(nil)
	_ plugin.HTTPProvider    = (*Module)(nil)
	_ plugin.HealthChecker   = (*Module)(nil)
	_ plugin.EventSubscriber = (*Module)(nil)
	_ roles.AnalyticsProvider = (*Module)(nil)
)

// Module implements the Insight analytics plugin.
type Module struct {
	logger  *zap.Logger
	cfg     InsightConfig
	store   *InsightStore
	bus     plugin.EventBus
	plugins plugin.PluginResolver

	mu        sync.RWMutex
	baselines map[string]struct{} // Tracked device:metric pairs

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new Insight plugin instance.
func New() *Module {
	return &Module{
		baselines: make(map[string]struct{}),
	}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "insight",
		Version:     "0.1.0",
		Description: "Statistical analytics and anomaly detection",
		Roles:       []string{roles.RoleAnalytics},
		Required:    false,
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger

	m.cfg = DefaultConfig()
	if deps.Config != nil {
		if err := deps.Config.Unmarshal(&m.cfg); err != nil {
			return fmt.Errorf("unmarshal insight config: %w", err)
		}
	}

	if deps.Store != nil {
		if err := deps.Store.Migrate(context.Background(), "insight", migrations()); err != nil {
			return fmt.Errorf("insight migrations: %w", err)
		}
		m.store = NewInsightStore(deps.Store.DB())
	}

	m.bus = deps.Bus
	m.plugins = deps.Plugins

	m.logger.Info("insight module initialized",
		zap.Float64("ewma_alpha", m.cfg.EWMAAlpha),
		zap.Float64("zscore_threshold", m.cfg.ZScoreThreshold),
		zap.Duration("learning_period", m.cfg.LearningPeriod),
	)
	return nil
}

func (m *Module) Start(_ context.Context) error {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.logger.Info("insight module started")
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	m.logger.Info("insight module stopped")
	return nil
}

// -- plugin.HealthChecker --

// Health implements plugin.HealthChecker.
func (m *Module) Health(_ context.Context) plugin.HealthStatus {
	m.mu.RLock()
	count := len(m.baselines)
	m.mu.RUnlock()

	details := map[string]string{
		"baselines_tracked": strconv.Itoa(count),
	}

	llmAvailable := "false"
	if m.plugins != nil {
		if providers := m.plugins.ResolveByRole(roles.RoleLLM); len(providers) > 0 {
			llmAvailable = "true"
		}
	}
	details["llm_available"] = llmAvailable

	return plugin.HealthStatus{
		Status:  "healthy",
		Details: details,
	}
}

// -- plugin.EventSubscriber --

// Subscriptions implements plugin.EventSubscriber.
func (m *Module) Subscriptions() []plugin.Subscription {
	return []plugin.Subscription{
		{Topic: TopicMetricsCollected, Handler: m.handleMetricsCollected},
		{Topic: TopicAlertTriggered, Handler: m.handleAlertTriggered},
		{Topic: TopicAlertResolved, Handler: m.handleAlertResolved},
		{Topic: TopicDeviceDiscovered, Handler: m.handleDeviceDiscovered},
	}
}

// Event handler stubs -- algorithms wired in PR 2.

func (m *Module) handleMetricsCollected(_ context.Context, event plugin.Event) {
	m.logger.Debug("received metrics event", zap.String("source", event.Source))
}

func (m *Module) handleAlertTriggered(_ context.Context, event plugin.Event) {
	m.logger.Debug("received alert triggered event", zap.String("source", event.Source))
}

func (m *Module) handleAlertResolved(_ context.Context, event plugin.Event) {
	m.logger.Debug("received alert resolved event", zap.String("source", event.Source))
}

func (m *Module) handleDeviceDiscovered(_ context.Context, event plugin.Event) {
	m.logger.Debug("received device discovered event", zap.String("source", event.Source))
}

// -- roles.AnalyticsProvider --

// Anomalies implements roles.AnalyticsProvider.
func (m *Module) Anomalies(ctx context.Context, deviceID string) ([]analytics.Anomaly, error) {
	if m.store == nil {
		return nil, nil
	}
	return m.store.ListAnomalies(ctx, deviceID, 100)
}

// Baselines implements roles.AnalyticsProvider.
func (m *Module) Baselines(ctx context.Context, deviceID string) ([]analytics.Baseline, error) {
	if m.store == nil {
		return nil, nil
	}
	return m.store.GetBaselines(ctx, deviceID)
}

// Forecasts implements roles.AnalyticsProvider.
func (m *Module) Forecasts(ctx context.Context, deviceID string) ([]analytics.Forecast, error) {
	if m.store == nil {
		return nil, nil
	}
	return m.store.GetForecasts(ctx, deviceID)
}
