package recon

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/HerbHall/netvantage/pkg/plugin"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin       = (*Module)(nil)
	_ plugin.HTTPProvider = (*Module)(nil)
)

// Module implements the Recon network discovery plugin.
type Module struct {
	logger *zap.Logger
	config plugin.Config
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
	m.config = deps.Config
	m.logger = deps.Logger
	m.logger.Info("recon module initialized")
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	m.logger.Info("recon module started")
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	m.logger.Info("recon module stopped")
	return nil
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "POST", Path: "/scan", Handler: m.handleScan},
		{Method: "GET", Path: "/scans", Handler: m.handleListScans},
	}
}

func (m *Module) handleScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "not_implemented",
		"message": "network scanning will be implemented in Phase 1",
	})
}

func (m *Module) handleListScans(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]any{})
}
