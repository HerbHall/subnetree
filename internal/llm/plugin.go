package llm

import (
	"context"
	"fmt"

	"github.com/HerbHall/subnetree/internal/llm/anthropic"
	"github.com/HerbHall/subnetree/internal/llm/ollama"
	"github.com/HerbHall/subnetree/internal/llm/openai"
	pkgllm "github.com/HerbHall/subnetree/pkg/llm"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/HerbHall/subnetree/pkg/roles"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ plugin.Plugin        = (*Module)(nil)
	_ plugin.HealthChecker = (*Module)(nil)
	_ plugin.HTTPProvider  = (*Module)(nil)
	_ roles.LLMProvider    = (*Module)(nil)
)

// CredentialDecrypter decrypts stored credentials by ID.
// Resolved from the plugin registry via RoleCredentialStore.
type CredentialDecrypter interface {
	DecryptCredentialData(ctx context.Context, id string) (map[string]any, error)
}

// ModuleConfig holds the LLM module configuration with per-provider sub-configs.
type ModuleConfig struct {
	Provider  string           `mapstructure:"provider"` // "ollama" (default), "openai", "anthropic"
	Ollama    ollama.Config    `mapstructure:"ollama"`
	OpenAI    openai.Config    `mapstructure:"openai"`
	Anthropic anthropic.Config `mapstructure:"anthropic"`
}

// Module implements the LLM plugin, wrapping a configurable provider.
type Module struct {
	logger   *zap.Logger
	provider pkgllm.Provider
	plugins  plugin.PluginResolver
	cfg      ModuleConfig
}

// New creates a new LLM plugin instance.
func New() *Module {
	return &Module{}
}

func (m *Module) Info() plugin.PluginInfo {
	return plugin.PluginInfo{
		Name:        "llm",
		Version:     "0.2.0",
		Description: "LLM provider integration (Ollama, OpenAI, Anthropic)",
		Roles:       []string{roles.RoleLLM},
		Required:    false,
		APIVersion:  plugin.APIVersionCurrent,
	}
}

func (m *Module) Init(_ context.Context, deps plugin.Dependencies) error {
	m.logger = deps.Logger
	m.plugins = deps.Plugins

	m.cfg = ModuleConfig{
		Provider:  "ollama",
		Ollama:    ollama.DefaultConfig(),
		OpenAI:    openai.DefaultConfig(),
		Anthropic: anthropic.DefaultConfig(),
	}

	if deps.Config != nil {
		if err := deps.Config.Unmarshal(&m.cfg); err != nil {
			return fmt.Errorf("unmarshal llm config: %w", err)
		}
	}

	provider, err := newProvider(m.cfg, m.plugins, m.logger)
	if err != nil {
		return fmt.Errorf("create %s provider: %w", m.cfg.Provider, err)
	}
	m.provider = provider

	m.logger.Info("llm plugin initialized",
		zap.String("provider", m.cfg.Provider),
	)
	return nil
}

func (m *Module) Start(ctx context.Context) error {
	hr, ok := m.provider.(pkgllm.HealthReporter)
	if !ok {
		return nil
	}

	if err := hr.Heartbeat(ctx); err != nil {
		m.logger.Warn("llm provider not reachable; features will be unavailable until it comes online",
			zap.String("provider", m.cfg.Provider),
			zap.Error(err),
		)
		return nil
	}

	models, err := hr.ListModels(ctx)
	if err != nil {
		m.logger.Warn("failed to list models",
			zap.String("provider", m.cfg.Provider),
			zap.Error(err),
		)
		return nil
	}

	m.logger.Info("llm provider connected",
		zap.String("provider", m.cfg.Provider),
		zap.Strings("models", models),
	)
	return nil
}

func (m *Module) Stop(_ context.Context) error {
	m.logger.Info("llm plugin stopped")
	return nil
}

// Health implements plugin.HealthChecker.
func (m *Module) Health(ctx context.Context) plugin.HealthStatus {
	hr, ok := m.provider.(pkgllm.HealthReporter)
	if !ok {
		return plugin.HealthStatus{Status: "healthy", Message: "no health reporter"}
	}

	if err := hr.Heartbeat(ctx); err != nil {
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

// Routes implements plugin.HTTPProvider.
func (m *Module) Routes() []plugin.Route {
	return []plugin.Route{
		{Method: "GET", Path: "/config", Handler: m.handleGetConfig},
		{Method: "PUT", Path: "/config", Handler: m.handlePutConfig},
		{Method: "POST", Path: "/test", Handler: m.handleTestConnection},
	}
}

// newProvider creates a provider based on the config.
func newProvider(cfg ModuleConfig, plugins plugin.PluginResolver, logger *zap.Logger) (pkgllm.Provider, error) {
	switch cfg.Provider {
	case "ollama", "":
		return ollama.New(cfg.Ollama, logger)

	case "openai":
		apiKey, err := resolveAPIKey(cfg.OpenAI.CredentialID, plugins)
		if err != nil {
			return nil, fmt.Errorf("resolve openai credentials: %w", err)
		}
		return openai.New(cfg.OpenAI, apiKey, logger)

	case "anthropic":
		apiKey, err := resolveAPIKey(cfg.Anthropic.CredentialID, plugins)
		if err != nil {
			return nil, fmt.Errorf("resolve anthropic credentials: %w", err)
		}
		return anthropic.New(cfg.Anthropic, apiKey, logger)

	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}

// resolveAPIKey retrieves an API key from the credential vault.
func resolveAPIKey(credentialID string, plugins plugin.PluginResolver) (string, error) {
	if credentialID == "" {
		return "", fmt.Errorf("credential_id is required for cloud providers")
	}

	if plugins == nil {
		return "", fmt.Errorf("plugin resolver not available")
	}

	vaultPlugins := plugins.ResolveByRole(roles.RoleCredentialStore)
	if len(vaultPlugins) == 0 {
		return "", fmt.Errorf("no credential store available; configure vault first")
	}

	decrypter, ok := vaultPlugins[0].(CredentialDecrypter)
	if !ok {
		return "", fmt.Errorf("credential store does not support decryption")
	}

	data, err := decrypter.DecryptCredentialData(context.Background(), credentialID)
	if err != nil {
		return "", fmt.Errorf("decrypt credential %s: %w", credentialID, err)
	}

	apiKey, ok := data["api_key"]
	if !ok {
		return "", fmt.Errorf("credential %s has no api_key field", credentialID)
	}

	keyStr, ok := apiKey.(string)
	if !ok {
		return "", fmt.Errorf("credential %s api_key is not a string", credentialID)
	}

	return keyStr, nil
}
