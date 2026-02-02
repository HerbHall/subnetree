package dispatch

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

// Module implements the Dispatch agent management plugin.
type Module struct {
	logger *zap.Logger
	config plugin.Config
}

// New creates a new Dispatch plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "dispatch",
		Version:     "0.1.0",
		Description: "Scout agent enrollment and management",
		Roles:       []string{"agent_management"},
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(ctx context.Context, deps plugin.Dependencies) error {
	m.config = deps.Config
	m.logger = deps.Logger
	m.logger.Info("dispatch module initialized")
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	m.logger.Info("dispatch module started")
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	m.logger.Info("dispatch module stopped")
	return nil
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/agents", Handler: m.handleListAgents},
		{Method: "GET", Path: "/agents/{id}", Handler: m.handleGetAgent},
	}
}

func (m *Module) handleListAgents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]any{})
}

func (m *Module) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "not_implemented",
		"message": "agent management will be implemented in Phase 1b",
	})
}
