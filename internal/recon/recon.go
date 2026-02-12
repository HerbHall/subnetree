package recon

import (
	"context"
	"strconv"
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

// Module implements the Recon network discovery plugin.
type Module struct {
	logger       *zap.Logger
	cfg          ReconConfig
	store        *ReconStore
	bus          plugin.EventBus
	oui          *OUITable
	orchestrator *ScanOrchestrator
	activeScans  sync.Map // scanID -> context.CancelFunc
	wg           sync.WaitGroup
	scanCtx      context.Context
	scanCancel   context.CancelFunc
}

// New creates a new Recon plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "recon",
		Version:     "0.1.0",
		Description: "Network discovery and device scanning",
		Roles:       []string{"discovery"},
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(ctx context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger
	m.bus = deps.Bus

	// Load config with defaults.
	m.cfg = DefaultConfig()
	if deps.Config != nil {
		if d := deps.Config.GetDuration("scan_timeout"); d > 0 {
			m.cfg.ScanTimeout = d
		}
		if d := deps.Config.GetDuration("ping_timeout"); d > 0 {
			m.cfg.PingTimeout = d
		}
		if v := deps.Config.GetInt("ping_count"); v > 0 {
			m.cfg.PingCount = v
		}
		if v := deps.Config.GetInt("concurrency"); v > 0 {
			m.cfg.Concurrency = v
		}
		if deps.Config.IsSet("arp_enabled") {
			m.cfg.ARPEnabled = deps.Config.GetBool("arp_enabled")
		}
		if d := deps.Config.GetDuration("device_lost_after"); d > 0 {
			m.cfg.DeviceLostAfter = d
		}
	}

	// Run database migrations.
	if err := deps.Store.Migrate(ctx, "recon", migrations()); err != nil {
		return err
	}

	// Initialize store and scanners.
	m.store = NewReconStore(deps.Store.DB())
	m.oui = NewOUITable()

	pinger := NewICMPScanner(m.cfg, m.logger.Named("icmp"))
	var arp ARPTableReader
	if m.cfg.ARPEnabled {
		arp = NewARPReader(m.logger.Named("arp"))
	}

	m.orchestrator = NewScanOrchestrator(m.store, m.bus, m.oui, pinger, arp, m.logger)

	m.logger.Info("recon module initialized")
	return nil
}

func (m *Module) Start(_ context.Context) error {
	m.scanCtx, m.scanCancel = context.WithCancel(context.Background())

	// Start device-lost checker background goroutine.
	m.wg.Add(1)
	go m.runDeviceLostChecker()

	m.logger.Info("recon module started")
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	m.logger.Info("recon module stopping, cancelling active scans")
	if m.scanCancel != nil {
		m.scanCancel()
	}
	// Cancel all individual scans.
	m.activeScans.Range(func(_, value any) bool {
		if cancel, ok := value.(context.CancelFunc); ok {
			cancel()
		}
		return true
	})
	m.wg.Wait()
	m.logger.Info("recon module stopped")
	return nil
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "POST", Path: "/scan", Handler: m.handleScan},
		{Method: "GET", Path: "/scans", Handler: m.handleListScans},
		{Method: "GET", Path: "/scans/{id}", Handler: m.handleGetScan},
		{Method: "GET", Path: "/topology", Handler: m.handleTopology},
		{Method: "GET", Path: "/devices", Handler: m.handleListDevices},
		{Method: "POST", Path: "/devices", Handler: m.handleCreateDevice},
		{Method: "GET", Path: "/devices/{id}", Handler: m.handleGetDevice},
		{Method: "PUT", Path: "/devices/{id}", Handler: m.handleUpdateDevice},
		{Method: "DELETE", Path: "/devices/{id}", Handler: m.handleDeleteDevice},
		{Method: "GET", Path: "/devices/{id}/history", Handler: m.handleDeviceHistory},
		{Method: "GET", Path: "/devices/{id}/scans", Handler: m.handleDeviceScans},
	}
}

// Health implements plugin.HealthChecker.
func (m *Module) Health(_ context.Context) plugin.HealthStatus {
	var activeCount int
	m.activeScans.Range(func(_, _ any) bool {
		activeCount++
		return true
	})

	details := map[string]string{
		"active_scans": strconv.Itoa(activeCount),
		"arp_enabled":  strconv.FormatBool(m.cfg.ARPEnabled),
	}

	return plugin.HealthStatus{
		Status:  "ok",
		Details: details,
	}
}

// runDeviceLostChecker periodically checks for devices that haven't been seen
// within the configured DeviceLostAfter threshold and marks them offline.
func (m *Module) runDeviceLostChecker() {
	defer m.wg.Done()

	interval := m.cfg.DeviceLostAfter / 4
	if interval < time.Minute {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	m.logger.Info("device lost checker started",
		zap.Duration("check_interval", interval),
		zap.Duration("device_lost_after", m.cfg.DeviceLostAfter),
	)

	for {
		select {
		case <-m.scanCtx.Done():
			m.logger.Info("device lost checker stopped")
			return
		case <-ticker.C:
			m.checkForLostDevices()
		}
	}
}

func (m *Module) checkForLostDevices() {
	ctx := m.scanCtx
	threshold := time.Now().Add(-m.cfg.DeviceLostAfter)

	stale, err := m.store.FindStaleDevices(ctx, threshold)
	if err != nil {
		m.logger.Error("failed to find stale devices", zap.Error(err))
		return
	}

	for i := range stale {
		if err := m.store.MarkDeviceOffline(ctx, stale[i].ID); err != nil {
			m.logger.Error("failed to mark device offline",
				zap.String("device_id", stale[i].ID),
				zap.Error(err),
			)
			continue
		}

		ip := ""
		if len(stale[i].IPAddresses) > 0 {
			ip = stale[i].IPAddresses[0]
		}

		m.publishEvent(ctx, TopicDeviceLost, DeviceLostEvent{
			DeviceID: stale[i].ID,
			IP:       ip,
			LastSeen: stale[i].LastSeen,
		})

		m.logger.Info("device marked as lost",
			zap.String("device_id", stale[i].ID),
			zap.String("ip", ip),
			zap.Time("last_seen", stale[i].LastSeen),
		)
	}
}

// publishEvent publishes an event to the event bus.
func (m *Module) publishEvent(ctx context.Context, topic string, payload any) {
	if m.bus == nil {
		return
	}
	m.bus.PublishAsync(ctx, plugin.Event{
		Topic:     topic,
		Source:    "recon",
		Timestamp: time.Now(),
		Payload:   payload,
	})
}

// newScanContext creates a child context from the module's scan context.
func (m *Module) newScanContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(m.scanCtx)
}
