// Package llm provides the public SDK types for LLM provider integrations.
// All LLM provider plugins (Ollama, OpenAI, Anthropic) implement these
// interfaces. Implementations live in internal/llm/{provider}/ adapters.
//
// This package is Apache 2.0 licensed, part of the public plugin SDK.
package llm

import "context"

// Provider is the core interface implemented by all LLM provider plugins.
// It exposes single-prompt generation and multi-turn chat completion.
type Provider interface {
	// Generate creates a completion from a single prompt.
	// Use CallOption values to override model, temperature, or enable streaming.
	Generate(ctx context.Context, prompt string, opts ...CallOption) (*Response, error)

	// Chat creates a completion from a conversation history.
	// Use CallOption values to override model, temperature, or enable streaming.
	Chat(ctx context.Context, messages []Message, opts ...CallOption) (*Response, error)
}

// HealthReporter is optionally implemented by providers that can report
// connection health and model availability. Detected via type assertion.
type HealthReporter interface {
	// Heartbeat checks whether the LLM service is reachable.
	Heartbeat(ctx context.Context) error

	// ListModels returns the names of models available from this provider.
	ListModels(ctx context.Context) ([]string, error)
}

// CallOption configures a single Generate or Chat call.
type CallOption func(*CallConfig)

// CallConfig holds the resolved configuration for a single LLM call.
// Users interact through CallOption functions, not this struct directly.
type CallConfig struct {
	Model       string
	Temperature float64
	MaxTokens   int
	StreamFunc  func(ctx context.Context, chunk []byte) error
}

// WithModel sets the model to use for this call, overriding the provider default.
func WithModel(model string) CallOption {
	return func(c *CallConfig) { c.Model = model }
}

// WithTemperature sets the sampling temperature.
// 0.0 = deterministic, 1.0+ = creative. Provider default if unset.
func WithTemperature(temp float64) CallOption {
	return func(c *CallConfig) { c.Temperature = temp }
}

// WithMaxTokens sets the maximum number of tokens to generate.
func WithMaxTokens(max int) CallOption {
	return func(c *CallConfig) { c.MaxTokens = max }
}

// WithStreamFunc enables streaming mode. The function is called for each
// chunk received from the provider. Return a non-nil error to abort streaming.
func WithStreamFunc(fn func(ctx context.Context, chunk []byte) error) CallOption {
	return func(c *CallConfig) { c.StreamFunc = fn }
}

// ApplyOptions creates a CallConfig from a list of options, starting from defaults.
func ApplyOptions(opts ...CallOption) CallConfig {
	cfg := CallConfig{
		Temperature: 0.7,
		MaxTokens:   2048,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}
