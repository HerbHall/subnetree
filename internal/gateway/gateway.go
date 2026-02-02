package gateway

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

// Module implements the Gateway remote access plugin.
type Module struct {
	logger *zap.Logger
	config plugin.Config
}

// New creates a new Gateway plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "gateway",
		Version:     "0.1.0",
		Description: "Remote access via Apache Guacamole",
		Roles:       []string{"remote_access"},
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(ctx context.Context, deps plugin.Dependencies) error {
	m.config = deps.Config
	m.logger = deps.Logger
	m.logger.Info("gateway module initialized")
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	m.logger.Info("gateway module started")
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	m.logger.Info("gateway module stopped")
	return nil
}

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/sessions", Handler: m.handleListSessions},
	}
}

func (m *Module) handleListSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode([]any{})
}
