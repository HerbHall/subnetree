package dispatch

import (
	"context"
	"fmt"

	"github.com/HerbHall/subnetree/pkg/plugin"
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
	cfg    DispatchConfig
	store  *DispatchStore
	bus    plugin.EventBus
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

func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.config = deps.Config
	m.logger = deps.Logger

	m.cfg = DefaultConfig()
	if deps.Config != nil {
		if err := deps.Config.Unmarshal(&m.cfg); err != nil {
			return fmt.Errorf("unmarshal dispatch config: %w", err)
		}
	}

	if deps.Store != nil {
		if err := deps.Store.Migrate(context.Background(), "dispatch", migrations()); err != nil {
			return fmt.Errorf("dispatch migrations: %w", err)
		}
		m.store = NewDispatchStore(deps.Store.DB())
	}

	m.bus = deps.Bus

	m.logger.Info("dispatch module initialized",
		zap.String("grpc_addr", m.cfg.GRPCAddr),
		zap.Duration("agent_timeout", m.cfg.AgentTimeout),
		zap.Duration("enrollment_token_expiry", m.cfg.EnrollmentTokenExpiry),
	)
	return nil
}

func (m *Module) Start(_ context.Context) error {
	m.logger.Info("dispatch module started")
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	m.logger.Info("dispatch module stopped")
	return nil
}

// Routes is implemented in handlers.go.
