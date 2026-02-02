package pulse

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

// Module implements the Pulse monitoring plugin.
type Module struct {
	logger *zap.Logger
	config plugin.Config
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
		Roles:       []string{"monitoring"},
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(ctx context.Context, deps plugin.Dependencies) error {
	m.config = deps.Config
	m.logger = deps.Logger
	m.logger.Info("pulse module initialized")
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	m.logger.Info("pulse module started")
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	m.logger.Info("pulse module stopped")
	return nil
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/status", Handler: m.handleStatus},
	}
}

func (m *Module) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "not_implemented",
		"message": "monitoring will be implemented in Phase 2",
	})
}
