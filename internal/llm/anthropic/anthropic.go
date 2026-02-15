package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/HerbHall/subnetree/pkg/llm"
	"go.uber.org/zap"
)

const baseURL = "https://api.anthropic.com"

// Compile-time interface guards.
var (
	_ llm.Provider       = (*Provider)(nil)
	_ llm.HealthReporter = (*Provider)(nil)
)

// Provider implements llm.Provider for Anthropic using its Messages API.
type Provider struct {
	apiKey     string
	httpClient *http.Client
	cfg        Config
	logger     *zap.Logger
}

// New creates an Anthropic provider.
func New(cfg Config, apiKey string, logger *zap.Logger) (*Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic: api key is required")
	}

	return &Provider{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: cfg.Timeout},
		cfg:        cfg,
		logger:     logger,
	}, nil
}

// Generate creates a completion from a single prompt.
func (p *Provider) Generate(ctx context.Context, prompt string, opts ...llm.CallOption) (*llm.Response, error) {
	return p.Chat(ctx, []llm.Message{{Role: "user", Content: prompt}}, opts...)
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

	req := messagesRequest{
		Model:       model,
		Messages:    apiMessages,
		MaxTokens:   cfg.MaxTokens,
		Temperature: cfg.Temperature,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal messages request: %w", err)
	}

	respBody, err := p.doPost(ctx, "/v1/messages", body)
	if err != nil {
		return nil, mapError(err)
	}
	defer respBody.Close()

	var resp messagesResponse
	if err := json.NewDecoder(respBody).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode messages response: %w", err)
	}

	var content strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			content.WriteString(block.Text)
		}
	}

	return &llm.Response{
		Content: content.String(),
		Model:   resp.Model,
		Usage: llm.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
		Done: true,
	}, nil
}

// Heartbeat checks whether the Anthropic API is reachable by listing models.
func (p *Provider) Heartbeat(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", http.NoBody)
	if err != nil {
		return mapError(err)
	}
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return mapError(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mapError(&anthropicStatusError{StatusCode: resp.StatusCode, Message: "heartbeat failed"})
	}
	return nil
}

// ListModels returns the available Anthropic model IDs.
func (p *Provider) ListModels(_ context.Context) ([]string, error) {
	return []string{
		"claude-sonnet-4-5-20250929",
		"claude-haiku-4-5-20251001",
		"claude-opus-4-6",
	}, nil
}

// doPost sends an authenticated POST request and returns the response body.
func (p *Provider) doPost(ctx context.Context, path string, body []byte) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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

// parseStatusError reads an error response body.
func parseStatusError(resp *http.Response) *anthropicStatusError {
	var errResp struct {
		Type  string `json:"type"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil || json.Unmarshal(data, &errResp) != nil {
		return &anthropicStatusError{StatusCode: resp.StatusCode, Message: resp.Status}
	}

	msg := errResp.Error.Message
	if msg == "" {
		msg = resp.Status
	}
	return &anthropicStatusError{
		StatusCode: resp.StatusCode,
		Type:       errResp.Error.Type,
		Message:    msg,
	}
}

// --- Anthropic Messages API types (internal) ---

type messagesRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Content []contentBlock `json:"content"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}
