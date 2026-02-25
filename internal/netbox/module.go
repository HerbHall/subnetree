package netbox

import (
	"context"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin       = (*Module)(nil)
	_ plugin.HTTPProvider = (*Module)(nil)
)

// DeviceReader provides read access to SubNetree device data.
// Implemented via an adapter in the composition root (main.go).
type DeviceReader interface {
	ListAllDevices(ctx context.Context) ([]models.Device, error)
	GetDevice(ctx context.Context, id string) (*models.Device, error)
}

// Module implements the NetBox CMDB export plugin.
// It syncs SubNetree device inventory to a NetBox instance via its REST API.
type Module struct {
	logger       *zap.Logger
	cfg          Config
	client       *Client
	deviceReader DeviceReader
}

// New creates a new NetBox plugin instance.
func New() *Module {
	return &Module{}
}

// SetDeviceReader injects the device reader. Called from the composition root
// (main.go) to wire the recon module's store without cross-internal imports.
func (m *Module) SetDeviceReader(r DeviceReader) {
	m.deviceReader = r
}

// Info returns the plugin metadata.
func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:         "netbox",
		Version:      "0.1.0",
		Description:  "NetBox CMDB export integration",
		Dependencies: []string{"recon"},
		APIVersion:   plugin.APIVersionCurrent,
	}
}

// Init initializes the module with its dependencies.
func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger
	m.cfg = DefaultConfig()

	if deps.Config != nil {
		if err := deps.Config.Unmarshal(&m.cfg); err != nil {
			m.logger.Warn("failed to unmarshal netbox config, using defaults", zap.Error(err))
		}
	}

	// Only create the client if URL and token are configured.
	if m.cfg.URL != "" && m.cfg.Token != "" {
		m.client = NewClient(m.cfg.URL, m.cfg.Token, m.cfg.Timeout)
		m.logger.Info("netbox client configured",
			zap.String("url", m.cfg.URL),
			zap.Bool("dry_run", m.cfg.DryRun),
		)
	} else {
		m.logger.Info("netbox module disabled (url or token not configured)")
	}

	m.logger.Info("netbox module initialized")
	return nil
}

// Start begins the module's operations. NetBox sync is stateless and on-demand,
// so there is nothing to start.
func (m *Module) Start(_ context.Context) error {
	m.logger.Info("netbox module started")
	return nil
}

// Stop gracefully shuts down the module.
func (m *Module) Stop(_ context.Context) error {
	m.logger.Info("netbox module stopped")
	return nil
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "POST", Path: "/sync", Handler: m.handleSync},
		{Method: "POST", Path: "/sync/{id}", Handler: m.handleSyncDevice},
		{Method: "GET", Path: "/status", Handler: m.handleStatus},
	}
}
