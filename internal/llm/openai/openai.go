package openai

import (
	"bufio"
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

const baseURL = "https://api.openai.com"

// Compile-time interface guards.
var (
	_ llm.Provider       = (*Provider)(nil)
	_ llm.HealthReporter = (*Provider)(nil)
)

// Provider implements llm.Provider for OpenAI using its chat completions API.
type Provider struct {
	apiKey     string
	httpClient *http.Client
	cfg        Config
	logger     *zap.Logger
}

// New creates an OpenAI provider.
func New(cfg Config, apiKey string, logger *zap.Logger) (*Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openai: api key is required")
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

	req := chatRequest{
		Model:       model,
		Messages:    apiMessages,
		Temperature: cfg.Temperature,
		MaxTokens:   cfg.MaxTokens,
		Stream:      false,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	respBody, err := p.doPost(ctx, "/v1/chat/completions", body)
	if err != nil {
		return nil, mapError(err)
	}
	defer respBody.Close()

	var resp chatResponse
	if err := json.NewDecoder(respBody).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode chat response: %w", err)
	}

	var content string
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	return &llm.Response{
		Content: content,
		Model:   resp.Model,
		Usage: llm.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
		Done: true,
	}, nil
}

// Heartbeat checks whether the OpenAI API is reachable.
func (p *Provider) Heartbeat(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", http.NoBody)
	if err != nil {
		return mapError(err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return mapError(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mapError(&openaiStatusError{StatusCode: resp.StatusCode, Message: "heartbeat failed"})
	}
	return nil
}

// ListModels returns the available model IDs.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", http.NoBody)
	if err != nil {
		return nil, mapError(err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

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

	names := make([]string, len(result.Data))
	for i := range result.Data {
		names[i] = result.Data[i].ID
	}
	return names, nil
}

// doPost sends an authenticated POST request and returns the response body.
func (p *Provider) doPost(ctx context.Context, path string, body []byte) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

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
func parseStatusError(resp *http.Response) *openaiStatusError {
	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	// Read a limited amount to avoid unbounded reads.
	limited := io.LimitReader(resp.Body, 1<<16)
	scanner := bufio.NewScanner(limited)
	var raw strings.Builder
	for scanner.Scan() {
		raw.WriteString(scanner.Text())
	}

	if err := json.Unmarshal([]byte(raw.String()), &errResp); err != nil {
		return &openaiStatusError{StatusCode: resp.StatusCode, Message: resp.Status}
	}

	msg := errResp.Error.Message
	if msg == "" {
		msg = resp.Status
	}
	return &openaiStatusError{
		StatusCode: resp.StatusCode,
		Type:       errResp.Error.Type,
		Message:    msg,
	}
}

// --- OpenAI REST API types (internal) ---

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type listResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}
