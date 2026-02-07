package ollama

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/HerbHall/subnetree/pkg/llm"
	"github.com/ollama/ollama/api"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ llm.Provider       = (*Provider)(nil)
	_ llm.HealthReporter = (*Provider)(nil)
)

// Provider implements llm.Provider for Ollama.
type Provider struct {
	client *api.Client
	cfg    Config
	logger *zap.Logger
}

// New creates an Ollama provider. It does not verify connectivity;
// call Heartbeat explicitly if you need an early health check.
func New(cfg Config, logger *zap.Logger) (*Provider, error) {
	base, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse ollama url %q: %w", cfg.URL, err)
	}

	httpClient := &http.Client{Timeout: cfg.Timeout}
	client := api.NewClient(base, httpClient)

	return &Provider{
		client: client,
		cfg:    cfg,
		logger: logger,
	}, nil
}

// Generate creates a completion from a single prompt.
func (p *Provider) Generate(ctx context.Context, prompt string, opts ...llm.CallOption) (*llm.Response, error) {
	cfg := llm.ApplyOptions(opts...)

	model := cfg.Model
	if model == "" {
		model = p.cfg.Model
	}

	noStream := false
	req := &api.GenerateRequest{
		Model:   model,
		Prompt:  prompt,
		Stream:  &noStream,
		Options: buildOptions(cfg),
	}

	var content strings.Builder
	var metrics api.Metrics
	var done bool

	if cfg.StreamFunc != nil {
		// Streaming: leave Stream nil (defaults to true).
		req.Stream = nil
	}

	err := p.client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		if resp.Response != "" {
			content.WriteString(resp.Response)
			if cfg.StreamFunc != nil {
				if sErr := cfg.StreamFunc(ctx, []byte(resp.Response)); sErr != nil {
					return sErr
				}
			}
		}
		if resp.Done {
			metrics = resp.Metrics
			done = true
		}
		return nil
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &llm.Response{
		Content: content.String(),
		Model:   model,
		Usage: llm.Usage{
			PromptTokens:     metrics.PromptEvalCount,
			CompletionTokens: metrics.EvalCount,
			TotalTokens:      metrics.PromptEvalCount + metrics.EvalCount,
		},
		Done: done,
	}, nil
}

// Chat creates a completion from a conversation history.
func (p *Provider) Chat(ctx context.Context, messages []llm.Message, opts ...llm.CallOption) (*llm.Response, error) {
	if len(messages) == 0 {
		return nil, llm.NewProviderError(llm.ErrCodeInvalidRequest, "messages must not be empty", nil)
	}

	cfg := llm.ApplyOptions(opts...)

	model := cfg.Model
	if model == "" {
		model = p.cfg.Model
	}

	apiMessages := make([]api.Message, len(messages))
	for i, m := range messages {
		apiMessages[i] = api.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	noStream := false
	req := &api.ChatRequest{
		Model:    model,
		Messages: apiMessages,
		Stream:   &noStream,
		Options:  buildOptions(cfg),
	}

	if cfg.StreamFunc != nil {
		req.Stream = nil
	}

	var content strings.Builder
	var metrics api.Metrics
	var done bool

	err := p.client.Chat(ctx, req, func(resp api.ChatResponse) error {
		if resp.Message.Content != "" {
			content.WriteString(resp.Message.Content)
			if cfg.StreamFunc != nil {
				if sErr := cfg.StreamFunc(ctx, []byte(resp.Message.Content)); sErr != nil {
					return sErr
				}
			}
		}
		if resp.Done {
			metrics = resp.Metrics
			done = true
		}
		return nil
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &llm.Response{
		Content: content.String(),
		Model:   model,
		Usage: llm.Usage{
			PromptTokens:     metrics.PromptEvalCount,
			CompletionTokens: metrics.EvalCount,
			TotalTokens:      metrics.PromptEvalCount + metrics.EvalCount,
		},
		Done: done,
	}, nil
}

// Heartbeat checks whether the Ollama server is reachable.
func (p *Provider) Heartbeat(ctx context.Context) error {
	return mapError(p.client.Heartbeat(ctx))
}

// ListModels returns the names of locally available models.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	resp, err := p.client.List(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	names := make([]string, len(resp.Models))
	for i, m := range resp.Models {
		names[i] = m.Name
	}
	return names, nil
}

// buildOptions converts CallConfig fields into Ollama's Options map.
func buildOptions(cfg llm.CallConfig) map[string]any {
	opts := make(map[string]any)
	if cfg.Temperature > 0 {
		opts["temperature"] = cfg.Temperature
	}
	if cfg.MaxTokens > 0 {
		opts["num_predict"] = cfg.MaxTokens
	}
	return opts
}
