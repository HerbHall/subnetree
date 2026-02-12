package docs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin        = (*Module)(nil)
	_ plugin.HTTPProvider  = (*Module)(nil)
	_ plugin.HealthChecker = (*Module)(nil)
)

// Module implements the Docs infrastructure documentation plugin.
type Module struct {
	logger     *zap.Logger
	cfg        Config
	store      *DocsStore
	bus        plugin.EventBus
	plugins    plugin.PluginResolver
	collectors []Collector
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// New creates a new Docs plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "docs",
		Version:     "0.1.0",
		Description: "Infrastructure documentation and configuration tracking",
		Required:    false,
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger

	m.cfg = DefaultConfig()
	if deps.Config != nil {
		if err := deps.Config.Unmarshal(&m.cfg); err != nil {
			return fmt.Errorf("unmarshal docs config: %w", err)
		}
	}

	if deps.Store != nil {
		if err := deps.Store.Migrate(context.Background(), "docs", migrations()); err != nil {
			return fmt.Errorf("docs migrations: %w", err)
		}
		m.store = NewStore(deps.Store.DB())
	}

	m.bus = deps.Bus
	m.plugins = deps.Plugins

	m.logger.Info("docs module initialized",
		zap.Duration("retention_period", m.cfg.RetentionPeriod),
		zap.Duration("collect_interval", m.cfg.CollectInterval),
		zap.Int("max_snapshots_per_app", m.cfg.MaxSnapshotsPerApp),
	)
	return nil
}

func (m *Module) Start(_ context.Context) error {
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.startMaintenance()
	m.logger.Info("docs module started")
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	if m.cancel != nil {
		m.cancel()
	}
	m.wg.Wait()
	m.logger.Info("docs module stopped")
	return nil
}

// Health implements plugin.HealthChecker.
func (m *Module) Health(_ context.Context) plugin.HealthStatus {
	details := map[string]string{}

	if m.store != nil {
		details["store"] = "connected"
	} else {
		details["store"] = "unavailable"
	}

	details["collectors"] = fmt.Sprintf("%d", len(m.collectors))

	status := "healthy"
	if m.store == nil {
		status = "degraded"
	}

	return plugin.HealthStatus{
		Status:  status,
		Details: details,
	}
}

// publishEvent publishes an event to the event bus.
func (m *Module) publishEvent(ctx context.Context, topic string, payload any) {
	if m.bus == nil {
		return
	}
	m.bus.PublishAsync(ctx, plugin.Event{
		Topic:     topic,
		Source:    "docs",
		Timestamp: time.Now(),
		Payload:   payload,
	})
}

// startMaintenance launches a background goroutine that periodically
// deletes old snapshots past the retention window.
func (m *Module) startMaintenance() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(m.cfg.MaintenanceInterval)
		defer ticker.Stop()

		for {
			select {
			case <-m.ctx.Done():
				return
			case <-ticker.C:
				m.runMaintenance()
			}
		}
	}()
}

// runMaintenance executes a single maintenance cycle.
func (m *Module) runMaintenance() {
	if m.store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-m.cfg.RetentionPeriod)

	deleted, err := m.store.DeleteOldSnapshots(ctx, cutoff)
	if err != nil {
		m.logger.Warn("failed to delete old snapshots", zap.Error(err))
	} else if deleted > 0 {
		m.logger.Info("purged old snapshots", zap.Int64("count", deleted))
	}
}
