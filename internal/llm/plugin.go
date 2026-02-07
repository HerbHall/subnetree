package llm

import (
	"context"
	"fmt"

	"github.com/HerbHall/subnetree/internal/llm/ollama"
	pkgllm "github.com/HerbHall/subnetree/pkg/llm"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/roles"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin        = (*Module)(nil)
	_ plugin.HealthChecker = (*Module)(nil)
	_ roles.LLMProvider    = (*Module)(nil)
)

// Module implements the LLM plugin, wrapping an Ollama provider.
type Module struct {
	logger   *zap.Logger
	provider *ollama.Provider
}

// New creates a new LLM plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "llm",
		Version:     "0.1.0",
		Description: "LLM provider integration (Ollama)",
		Roles:       []string{roles.RoleLLM},
		Required:    false,
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger

	cfg := ollama.DefaultConfig()
	if deps.Config != nil {
		if err := deps.Config.Unmarshal(&cfg); err != nil {
			return fmt.Errorf("unmarshal llm config: %w", err)
		}
	}

	provider, err := ollama.New(cfg, m.logger)
	if err != nil {
		return fmt.Errorf("create ollama provider: %w", err)
	}
	m.provider = provider

	m.logger.Info("llm plugin initialized",
		zap.String("provider", "ollama"),
		zap.String("url", cfg.URL),
		zap.String("model", cfg.Model),
	)
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	if err := m.provider.Heartbeat(ctx); err != nil {
		m.logger.Warn("ollama not reachable; LLM features will be unavailable until it comes online",
			zap.Error(err),
		)
		return nil
	}

	models, err := m.provider.ListModels(ctx)
	if err != nil {
		m.logger.Warn("failed to list ollama models", zap.Error(err))
		return nil
	}

	m.logger.Info("ollama connected", zap.Strings("models", models))
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	m.logger.Info("llm plugin stopped")
	return nil
}

// Health implements plugin.HealthChecker.
func (m *Module) Health(ctx context.Context) plugin.HealthStatus {
	if err := m.provider.Heartbeat(ctx); err != nil {
		return plugin.HealthStatus{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}
	return plugin.HealthStatus{Status: "healthy"}
}

// Provider implements roles.LLMProvider.
func (m *Module) Provider() pkgllm.Provider {
	return m.provider
}
