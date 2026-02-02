package vault

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

// Module implements the Vault credential management plugin.
type Module struct {
	logger *zap.Logger
	config plugin.Config
}

// New creates a new Vault plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "vault",
		Version:     "0.1.0",
		Description: "Credential storage and management",
		Roles:       []string{"credential_store"},
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(ctx context.Context, deps plugin.Dependencies) error {
	m.config = deps.Config
	m.logger = deps.Logger
	m.logger.Info("vault module initialized")
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	m.logger.Info("vault module started")
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	m.logger.Info("vault module stopped")
	return nil
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/credentials", Handler: m.handleListCredentials},
		{Method: "POST", Path: "/credentials", Handler: m.handleCreateCredential},
	}
}

func (m *Module) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]any{})
}

func (m *Module) handleCreateCredential(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "not_implemented",
		"message": "credential storage will be implemented in Phase 3",
	})
}
