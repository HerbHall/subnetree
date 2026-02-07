package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/HerbHall/subnetree/pkg/llm"
	"go.uber.org/zap"
)

// Compile-time interface guards.
var (
	_ llm.Provider       = (*Provider)(nil)
	_ llm.HealthReporter = (*Provider)(nil)
)

// Provider implements llm.Provider for Ollama using its REST API.
type Provider struct {
	baseURL    string
	httpClient *http.Client
	cfg        Config
	logger     *zap.Logger
}

// New creates an Ollama provider. It does not verify connectivity;
// call Heartbeat explicitly if you need an early health check.
func New(cfg Config, logger *zap.Logger) (*Provider, error) {
	if _, err := url.Parse(cfg.URL); err != nil {
		return nil, fmt.Errorf("parse ollama url %q: %w", cfg.URL, err)
	}

	return &Provider{
		baseURL:    strings.TrimRight(cfg.URL, "/"),
		httpClient: &http.Client{Timeout: cfg.Timeout},
		cfg:        cfg,
		logger:     logger,
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
	req := generateRequest{
		Model:   model,
		Prompt:  prompt,
		Stream:  &noStream,
		Options: buildOptions(cfg),
	}

	if cfg.StreamFunc != nil {
		req.Stream = nil // omit → Ollama defaults to streaming
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal generate request: %w", err)
	}

	respBody, err := p.doPost(ctx, "/api/generate", body)
	if err != nil {
		return nil, mapError(err)
	}
	defer respBody.Close()

	var content strings.Builder
	var metrics responseMetrics
	var done bool

	scanner := bufio.NewScanner(respBody)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var chunk generateResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		if chunk.Response != "" {
			content.WriteString(chunk.Response)
			if cfg.StreamFunc != nil {
				if sErr := cfg.StreamFunc(ctx, []byte(chunk.Response)); sErr != nil {
					return nil, sErr
				}
			}
		}
		if chunk.Done {
			metrics = chunk.responseMetrics
			done = true
		}
	}
	if err := scanner.Err(); err != nil {
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

	apiMessages := make([]chatMessage, len(messages))
	for i, m := range messages {
		apiMessages[i] = chatMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	noStream := false
	req := chatRequest{
		Model:    model,
		Messages: apiMessages,
		Stream:   &noStream,
		Options:  buildOptions(cfg),
	}

	if cfg.StreamFunc != nil {
		req.Stream = nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	respBody, err := p.doPost(ctx, "/api/chat", body)
	if err != nil {
		return nil, mapError(err)
	}
	defer respBody.Close()

	var content strings.Builder
	var metrics responseMetrics
	var done bool

	scanner := bufio.NewScanner(respBody)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var chunk chatResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		if chunk.Message.Content != "" {
			content.WriteString(chunk.Message.Content)
			if cfg.StreamFunc != nil {
				if sErr := cfg.StreamFunc(ctx, []byte(chunk.Message.Content)); sErr != nil {
					return nil, sErr
				}
			}
		}
		if chunk.Done {
			metrics = chunk.responseMetrics
			done = true
		}
	}
	if err := scanner.Err(); err != nil {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/", http.NoBody)
	if err != nil {
		return mapError(err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return mapError(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mapError(&ollamaStatusError{StatusCode: resp.StatusCode, Message: "heartbeat failed"})
	}
	return nil
}

// ListModels returns the names of locally available models.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/api/tags", http.NoBody)
	if err != nil {
		return nil, mapError(err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, mapError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mapError(parseStatusError(resp))
	}

	var result listResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode list response: %w", err)
	}

	names := make([]string, len(result.Models))
	for i := range result.Models {
		names[i] = result.Models[i].Name
	}
	return names, nil
}

// doPost sends a POST request and returns the response body.
// The caller must close the returned body.
func (p *Provider) doPost(ctx context.Context, path string, body []byte) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, parseStatusError(resp)
	}

	return resp.Body, nil
}

// parseStatusError reads an error response body and returns an ollamaStatusError.
func parseStatusError(resp *http.Response) *ollamaStatusError {
	var errResp struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return &ollamaStatusError{StatusCode: resp.StatusCode, Message: resp.Status}
	}
	msg := errResp.Error
	if msg == "" {
		msg = resp.Status
	}
	return &ollamaStatusError{StatusCode: resp.StatusCode, Message: msg}
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

// --- Ollama REST API types (internal) ---

type generateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  *bool          `json:"stream,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

type chatRequest struct {
	Model    string         `json:"model"`
	Messages []chatMessage  `json:"messages"`
	Stream   *bool          `json:"stream,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseMetrics struct {
	PromptEvalCount int `json:"prompt_eval_count"`
	EvalCount       int `json:"eval_count"`
}

type generateResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
	responseMetrics
}

type chatResponse struct {
	Model   string      `json:"model"`
	Message chatMessage `json:"message"`
	Done    bool        `json:"done"`
	responseMetrics
}

type listResponse struct {
	Models []listModel `json:"models"`
}

type listModel struct {
	Name string `json:"name"`
}
