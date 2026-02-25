package insight

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/HerbHall/subnetree/internal/insight/anomaly"
	"github.com/HerbHall/subnetree/internal/insight/forecast"
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
	states  *stateManager

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
	m.states = newStateManager(
		m.cfg.EWMAAlpha, m.cfg.CUSUMDrift, m.cfg.CUSUMThreshold,
		m.cfg.HWAlpha, m.cfg.HWBeta, m.cfg.HWGamma, m.cfg.HWSeasonLen,
	)

	m.logger.Info("insight module initialized",
		zap.Float64("ewma_alpha", m.cfg.EWMAAlpha),
		zap.Float64("zscore_threshold", m.cfg.ZScoreThreshold),
		zap.Duration("learning_period", m.cfg.LearningPeriod),
		zap.Float64("hw_alpha", m.cfg.HWAlpha),
		zap.Float64("hw_beta", m.cfg.HWBeta),
		zap.Float64("hw_gamma", m.cfg.HWGamma),
		zap.Int("hw_season_len", m.cfg.HWSeasonLen),
	)
	return nil
}

func (m *Module) Start(_ context.Context) error {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.startMaintenance()
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
	count := 0
	if m.states != nil {
		count = m.states.count()
	}

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

// -- Event Handlers --

// handleMetricsCollected is the main analytics pipeline entry point.
// It processes incoming metric points through the EWMA, Z-score, and CUSUM algorithms.
func (m *Module) handleMetricsCollected(_ context.Context, event plugin.Event) {
	points, ok := event.Payload.([]analytics.MetricPoint)
	if !ok {
		m.logger.Debug("ignored metrics event: unexpected payload type",
			zap.String("source", event.Source))
		return
	}

	for i := range points {
		m.processMetric(&points[i])
	}
}

// processMetric runs a single metric point through the analytics pipeline.
// When Holt-Winters has accumulated enough seasonal data (>= 2 * season_len),
// it is used for anomaly detection via expected range. Otherwise, EWMA + Z-score is used.
func (m *Module) processMetric(p *analytics.MetricPoint) {
	// Store raw metric for regression
	if m.store != nil {
		ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
		defer cancel()
		if err := m.store.InsertMetric(ctx, p); err != nil {
			m.logger.Warn("failed to store metric", zap.Error(err))
		}
	}

	// Get or create in-memory state
	state := m.states.getOrCreate(p.DeviceID, p.MetricName)

	// Track in baselines map
	m.mu.Lock()
	m.baselines[stateKey(p.DeviceID, p.MetricName)] = struct{}{}
	m.mu.Unlock()

	// Update EWMA baseline
	prevMean := state.EWMA.Mean
	prevStdDev := state.EWMA.StdDev()
	state.EWMA.Update(p.Value)

	// Update Holt-Winters model
	state.HoltWinters.Update(p.Value)

	// Skip anomaly detection during learning period
	if state.EWMA.Samples < m.cfg.MinSamplesStable {
		return
	}

	// Holt-Winters seasonal anomaly detection: use when model has seen >= 2 full seasons
	hwUsed := false
	if state.HoltWinters.IsInitialized() && state.HoltWinters.Samples >= 2*m.cfg.HWSeasonLen {
		lower, upper := state.HoltWinters.ExpectedRange(m.cfg.HWConfidence)
		if lower != upper && (p.Value < lower || p.Value > upper) {
			hwFitted := state.HoltWinters.Fitted()
			deviation := p.Value - hwFitted
			severity := anomaly.SeverityWarning
			if p.Value < lower-(upper-lower) || p.Value > upper+(upper-lower) {
				severity = anomaly.SeverityCritical
			}
			m.recordAnomaly(p, "holt_winters", severity, deviation, hwFitted)
			hwUsed = true
		}
	}

	// Fall back to Z-score when Holt-Winters did not flag (or is not ready)
	if !hwUsed {
		zResult := anomaly.ZScoreCheck(p.Value, prevMean, prevStdDev, m.cfg.ZScoreThreshold)
		if zResult.IsAnomaly {
			m.recordAnomaly(p, "zscore", zResult.Severity, zResult.ZScore, prevMean)
		}
	}

	// CUSUM check (normalized input) -- always runs for change-point detection
	if prevStdDev > 0 {
		normalized := (p.Value - prevMean) / prevStdDev
		cResult := state.CUSUM.Update(normalized)
		if cResult.IsChangePoint {
			severity := anomaly.SeverityWarning
			m.recordAnomaly(p, "cusum", severity, cResult.CUSUMHigh+cResult.CUSUMLow, prevMean)
		}
	}

	// Periodically run regression forecast (every 100 samples)
	if state.EWMA.Samples%100 == 0 && m.store != nil {
		m.runForecast(p.DeviceID, p.MetricName)
	}
}

// recordAnomaly stores an anomaly and publishes an event.
func (m *Module) recordAnomaly(p *analytics.MetricPoint, anomalyType, severity string, deviation, expected float64) {
	a := &analytics.Anomaly{
		ID:          fmt.Sprintf("%s:%s:%d", p.DeviceID, p.MetricName, time.Now().UnixNano()),
		DeviceID:    p.DeviceID,
		MetricName:  p.MetricName,
		Severity:    severity,
		Type:        anomalyType,
		Value:       p.Value,
		Expected:    expected,
		Deviation:   deviation,
		DetectedAt:  time.Now(),
		Description: fmt.Sprintf("%s anomaly on %s: value=%.2f expected=%.2f deviation=%.2f", anomalyType, p.MetricName, p.Value, expected, deviation),
	}

	m.logger.Info("anomaly detected",
		zap.String("device_id", p.DeviceID),
		zap.String("metric", p.MetricName),
		zap.String("type", anomalyType),
		zap.String("severity", severity),
		zap.Float64("value", p.Value),
		zap.Float64("expected", expected),
	)

	if m.store != nil {
		ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
		defer cancel()
		if err := m.store.InsertAnomaly(ctx, a); err != nil {
			m.logger.Warn("failed to store anomaly", zap.Error(err))
		}
	}

	if m.bus != nil {
		m.bus.PublishAsync(m.ctx, plugin.Event{
			Topic:   TopicAnomalyDetected,
			Source:  "insight",
			Payload: a,
		})
	}
}

// runForecast performs linear regression on recent metric data and stores the forecast.
func (m *Module) runForecast(deviceID, metricName string) {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	since := time.Now().Add(-m.cfg.ForecastWindow)
	points, err := m.store.GetMetricWindow(ctx, deviceID, metricName, since)
	if err != nil || len(points) < 2 {
		return
	}

	timestamps := make([]time.Time, len(points))
	values := make([]float64, len(points))
	for i, p := range points {
		timestamps[i] = p.Timestamp
		values[i] = p.Value
	}

	hours := forecast.TimeToHours(timestamps)
	// Use a default threshold of 90% (configurable per-metric in the future)
	result := forecast.LinearRegression(hours, values, 90.0)
	if result == nil {
		return
	}

	f := &analytics.Forecast{
		DeviceID:       deviceID,
		MetricName:     metricName,
		CurrentValue:   values[len(values)-1],
		PredictedValue: result.Predicted,
		Threshold:      90.0,
		Slope:          result.Slope,
		Confidence:     result.RSquared,
		GeneratedAt:    time.Now(),
	}
	if result.TimeToLimit != nil {
		d := *result.TimeToLimit
		f.TimeToThreshold = &d
	}

	if err := m.store.UpsertForecast(ctx, f); err != nil {
		m.logger.Warn("failed to store forecast", zap.Error(err))
	}

	// Publish warning if threshold will be reached within forecast window
	if result.TimeToLimit != nil && *result.TimeToLimit < m.cfg.ForecastWindow && m.bus != nil {
		m.bus.PublishAsync(m.ctx, plugin.Event{
			Topic:   TopicForecastWarning,
			Source:  "insight",
			Payload: f,
		})
	}
}

func (m *Module) handleAlertTriggered(_ context.Context, event plugin.Event) {
	m.logger.Debug("received alert triggered event", zap.String("source", event.Source))
	// Alert correlation will use topology data from Recon when Pulse ships.
	// For now, store the alert context for future correlation.
}

func (m *Module) handleAlertResolved(_ context.Context, event plugin.Event) {
	m.logger.Debug("received alert resolved event", zap.String("source", event.Source))
}

func (m *Module) handleDeviceDiscovered(_ context.Context, event plugin.Event) {
	data, ok := event.Payload.(json.RawMessage)
	if !ok {
		m.logger.Debug("ignored device event: unexpected payload type",
			zap.String("source", event.Source))
		return
	}
	m.logger.Debug("device discovered, topology updated",
		zap.String("source", event.Source),
		zap.Int("payload_bytes", len(data)))
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
