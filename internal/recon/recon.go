package recon

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

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
)

// Module implements the Recon network discovery plugin.
type Module struct {
	logger        *zap.Logger
	cfg           ReconConfig
	store         *ReconStore
	bus           plugin.EventBus
	oui           *OUITable
	orchestrator  *ScanOrchestrator
	snmpCollector *SNMPCollector
	mdns          *MDNSListener
	upnp          *UPNPDiscoverer
	scheduler     *ScanScheduler
	consolidator  *ScanConsolidator
	credAccessor  CredentialAccessor
	credProvider  roles.CredentialProvider
	profileSource ProfileSource
	activeScans   sync.Map // scanID -> context.CancelFunc
	wg            sync.WaitGroup
	scanCtx       context.Context
	scanCancel    context.CancelFunc
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
		if deps.Config.IsSet("mdns_enabled") {
			m.cfg.MDNSEnabled = deps.Config.GetBool("mdns_enabled")
		}
		if d := deps.Config.GetDuration("mdns_interval"); d > 0 {
			m.cfg.MDNSInterval = d
		}
		if deps.Config.IsSet("upnp_enabled") {
			m.cfg.UPNPEnabled = deps.Config.GetBool("upnp_enabled")
		}
		if d := deps.Config.GetDuration("upnp_interval"); d > 0 {
			m.cfg.UPNPInterval = d
		}
		if deps.Config.IsSet("schedule.enabled") {
			m.cfg.Schedule.Enabled = deps.Config.GetBool("schedule.enabled")
		}
		if d := deps.Config.GetDuration("schedule.interval"); d > 0 {
			m.cfg.Schedule.Interval = d
		}
		if v := deps.Config.GetString("schedule.quiet_start"); v != "" {
			m.cfg.Schedule.QuietStart = v
		}
		if v := deps.Config.GetString("schedule.quiet_end"); v != "" {
			m.cfg.Schedule.QuietEnd = v
		}
		if v := deps.Config.GetString("schedule.subnet"); v != "" {
			m.cfg.Schedule.Subnet = v
		}
	}

	// Allow disabling discovery via environment for QC/testing containers.
	// Viper's Sub() does not inherit AutomaticEnv, so plugin-scoped env vars
	// like NV_RECON_MDNS_ENABLED are not visible to the sub-Viper. We check
	// them explicitly here.
	if v := os.Getenv("NV_RECON_MDNS_ENABLED"); strings.EqualFold(v, "false") {
		m.cfg.MDNSEnabled = false
	}
	if v := os.Getenv("NV_RECON_UPNP_ENABLED"); strings.EqualFold(v, "false") {
		m.cfg.UPNPEnabled = false
	}
	if v := os.Getenv("NV_RECON_SCHEDULE_ENABLED"); strings.EqualFold(v, "false") {
		m.cfg.Schedule.Enabled = false
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

	// Initialize mDNS listener if enabled.
	if m.cfg.MDNSEnabled {
		m.mdns = NewMDNSListener(m.store, m.bus, m.logger.Named("mdns"), m.cfg.MDNSInterval)
	}

	// Initialize UPnP discoverer if enabled.
	if m.cfg.UPNPEnabled {
		m.upnp = NewUPNPDiscoverer(m.store, m.bus, m.logger.Named("upnp"), m.cfg.UPNPInterval)
	}

	m.logger.Info("recon module initialized")
	return nil
}

func (m *Module) Start(_ context.Context) error {
	m.scanCtx, m.scanCancel = context.WithCancel(context.Background())

	// Initialize SNMP collector and wire it into the scan orchestrator
	// for FDB table walks during post-scan processing.
	m.snmpCollector = NewSNMPCollector(m.logger.Named("snmp"))
	m.orchestrator.SetSNMPWalker(m.snmpCollector)
	m.orchestrator.SetCredentialLookup(m)

	// Start device-lost checker background goroutine.
	m.wg.Add(1)
	go m.runDeviceLostChecker()

	// Start mDNS listener background goroutine if configured.
	if m.mdns != nil {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.mdns.Run(m.scanCtx)
		}()
		m.logger.Info("mDNS passive discovery enabled",
			zap.Duration("interval", m.cfg.MDNSInterval),
		)
	}

	// Start UPnP discoverer background goroutine if configured.
	if m.upnp != nil {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.upnp.Run(m.scanCtx)
		}()
		m.logger.Info("UPnP/SSDP discovery enabled",
			zap.Duration("interval", m.cfg.UPNPInterval),
		)
	}

	// Start scan scheduler if enabled.
	if m.cfg.Schedule.Enabled && m.cfg.Schedule.Subnet != "" {
		m.scheduler = NewScanScheduler(
			m.cfg.Schedule,
			m.orchestrator,
			m.store,
			&m.activeScans,
			&m.wg,
			m.newScanContext,
			m.logger.Named("scheduler"),
		)
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.scheduler.Run(m.scanCtx)
		}()
		m.logger.Info("scan scheduler enabled",
			zap.Duration("interval", m.cfg.Schedule.Interval),
			zap.String("subnet", m.cfg.Schedule.Subnet),
		)
	}

	// Start scan metrics consolidator background goroutine.
	m.consolidator = NewScanConsolidator(m.store, m.logger.Named("consolidation"))
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.consolidator.Run(m.scanCtx)
	}()

	m.logger.Info("recon module started")
	return nil
}

// Store returns the module's ReconStore for direct database operations.
// Used by the seed data system to populate demo data.
func (m *Module) Store() *ReconStore {
	return m.store
}

// SetCredentialAccessor sets the credential accessor used for SNMP discovery.
// Called from the composition root after all plugins are initialized.
func (m *Module) SetCredentialAccessor(ca CredentialAccessor) {
	m.credAccessor = ca
	if m.orchestrator != nil {
		m.orchestrator.SetCredentialAccessor(ca)
	}
}

// SetProfileSource sets the hardware profile source for bridging dispatch -> recon.
// Called from the composition root after all plugins are initialized.
func (m *Module) SetProfileSource(ps ProfileSource) {
	m.profileSource = ps
}

// FindSNMPCredentialForDevice implements CredentialLookup by delegating to the
// Module's credProvider (roles.CredentialProvider).
func (m *Module) FindSNMPCredentialForDevice(ctx context.Context, deviceID string) (string, error) {
	if m.credProvider == nil {
		return "", nil
	}
	creds, err := m.credProvider.CredentialsForDevice(ctx, deviceID)
	if err != nil {
		return "", err
	}
	for i := range creds {
		if creds[i].Type == "snmp_v2c" || creds[i].Type == "snmp_v3" {
			return creds[i].ID, nil
		}
	}
	return "", nil
}

func (m *Module) Stop(_ context.Context) error {
	m.logger.Info("recon module stopping, cancelling active scans")
	if m.scheduler != nil {
		m.scheduler.Stop()
	}
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

// Subscriptions implements plugin.EventSubscriber.
func (m *Module) Subscriptions() []plugin.Subscription {
	return []plugin.Subscription{
		{Topic: "dispatch.device.profiled", Handler: m.handleDeviceProfiled},
	}
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "POST", Path: "/scan", Handler: m.handleScan},
		{Method: "GET", Path: "/scans", Handler: m.handleListScans},
		{Method: "GET", Path: "/scans/{id}", Handler: m.handleGetScan},
		{Method: "GET", Path: "/scans/{id}/metrics", Handler: m.handleGetScanMetrics},
		{Method: "GET", Path: "/topology", Handler: m.handleTopology},
		{Method: "GET", Path: "/hierarchy", Handler: m.handleGetHierarchy},
		{Method: "GET", Path: "/topology/layouts", Handler: m.handleListTopologyLayouts},
		{Method: "POST", Path: "/topology/layouts", Handler: m.handleCreateTopologyLayout},
		{Method: "PUT", Path: "/topology/layouts/{id}", Handler: m.handleUpdateTopologyLayout},
		{Method: "DELETE", Path: "/topology/layouts/{id}", Handler: m.handleDeleteTopologyLayout},
		{Method: "GET", Path: "/devices", Handler: m.handleListDevices},
		{Method: "POST", Path: "/devices", Handler: m.handleCreateDevice},
		{Method: "GET", Path: "/devices/export", Handler: m.handleExportCSV},
		{Method: "POST", Path: "/devices/import", Handler: m.handleImportCSV},
		{Method: "GET", Path: "/devices/{id}", Handler: m.handleGetDevice},
		{Method: "PUT", Path: "/devices/{id}", Handler: m.handleUpdateDevice},
		{Method: "DELETE", Path: "/devices/{id}", Handler: m.handleDeleteDevice},
		{Method: "GET", Path: "/devices/{id}/history", Handler: m.handleDeviceHistory},
		{Method: "GET", Path: "/devices/{id}/scans", Handler: m.handleDeviceScans},
		{Method: "GET", Path: "/inventory/summary", Handler: m.handleInventorySummary},
		{Method: "PATCH", Path: "/devices/bulk", Handler: m.handleBulkUpdateDevices},
		{Method: "GET", Path: "/metrics/health-score", Handler: m.handleHealthScore},
		{Method: "GET", Path: "/metrics/aggregates", Handler: m.handleListMetricsAggregates},
		{Method: "GET", Path: "/metrics/raw", Handler: m.handleListRawMetrics},
		{Method: "GET", Path: "/movements", Handler: m.handleListServiceMovements},
		{Method: "POST", Path: "/snmp/discover", Handler: m.handleSNMPDiscover},
		{Method: "GET", Path: "/snmp/system/{device_id}", Handler: m.handleSNMPSystemInfo},
		{Method: "GET", Path: "/snmp/interfaces/{device_id}", Handler: m.handleSNMPInterfaces},
		{Method: "POST", Path: "/traceroute", Handler: m.handleTraceroute},
		{Method: "POST", Path: "/diag/ping", Handler: m.handleDiagPing},
		{Method: "POST", Path: "/diag/dns", Handler: m.handleDiagDNS},
		{Method: "POST", Path: "/diag/port-check", Handler: m.handleDiagPortCheck},
		{Method: "GET", Path: "/devices/{id}/hardware", Handler: m.handleGetDeviceHardware},
		{Method: "PUT", Path: "/devices/{id}/hardware", Handler: m.handleUpdateDeviceHardware},
		{Method: "GET", Path: "/devices/{id}/storage", Handler: m.handleGetDeviceStorage},
		{Method: "GET", Path: "/devices/{id}/gpu", Handler: m.handleGetDeviceGPU},
		{Method: "GET", Path: "/devices/{id}/services", Handler: m.handleGetDeviceServices},
		{Method: "GET", Path: "/inventory/hardware-summary", Handler: m.handleHardwareSummary},
		{Method: "GET", Path: "/devices/query/hardware", Handler: m.handleQueryDevicesByHardware},
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
		"mdns_enabled": strconv.FormatBool(m.cfg.MDNSEnabled),
		"upnp_enabled": strconv.FormatBool(m.cfg.UPNPEnabled),
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
