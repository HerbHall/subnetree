package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/HerbHall/subnetree/pkg/llm"
	"github.com/ollama/ollama/api"
	"go.uber.org/zap"
)

// newTestProvider creates a Provider pointing at the given httptest server URL.
func newTestProvider(t *testing.T, serverURL string) *Provider {
	t.Helper()
	base, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	return &Provider{
		client: api.NewClient(base, &http.Client{Timeout: 10 * time.Second}),
		cfg: Config{
			URL:     serverURL,
			Model:   "test-model",
			Timeout: 10 * time.Second,
		},
		logger: zap.NewNop(),
	}
}

// mockOllama returns an httptest server that handles Ollama API endpoints.
func mockOllama(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// Heartbeat
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ollama is running")) //nolint:errcheck
	})

	// Generate (non-streaming)
	mux.HandleFunc("POST /api/generate", func(w http.ResponseWriter, r *http.Request) {
		var req api.GenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := api.GenerateResponse{
			Model:    req.Model,
			Response: "Hello from Ollama!",
			Done:     true,
			Metrics: api.Metrics{
				PromptEvalCount: 5,
				EvalCount:       4,
				TotalDuration:   100 * time.Millisecond,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})

	// Chat (non-streaming)
	mux.HandleFunc("POST /api/chat", func(w http.ResponseWriter, r *http.Request) {
		var req api.ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp := api.ChatResponse{
			Model: req.Model,
			Message: api.Message{
				Role:    "assistant",
				Content: "The answer is 4.",
			},
			Done: true,
			Metrics: api.Metrics{
				PromptEvalCount: 10,
				EvalCount:       6,
				TotalDuration:   150 * time.Millisecond,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})

	// List models
	mux.HandleFunc("GET /api/tags", func(w http.ResponseWriter, r *http.Request) {
		resp := api.ListResponse{
			Models: []api.ListModelResponse{
				{Name: "qwen2.5:32b", Model: "qwen2.5:32b", Size: 1024},
				{Name: "llama3:8b", Model: "llama3:8b", Size: 512},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestNew_InvalidURL(t *testing.T) {
	_, err := New(Config{URL: "://bad"}, zap.NewNop())
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestGenerate_Success(t *testing.T) {
	srv := mockOllama(t)
	p := newTestProvider(t, srv.URL)

	resp, err := p.Generate(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.Content != "Hello from Ollama!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello from Ollama!")
	}
	if resp.Model != "test-model" {
		t.Errorf("Model = %q, want %q", resp.Model, "test-model")
	}
	if !resp.Done {
		t.Error("Done = false, want true")
	}
	if resp.Usage.PromptTokens != 5 {
		t.Errorf("PromptTokens = %d, want 5", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 4 {
		t.Errorf("CompletionTokens = %d, want 4", resp.Usage.CompletionTokens)
	}
	if resp.Usage.TotalTokens != 9 {
		t.Errorf("TotalTokens = %d, want 9", resp.Usage.TotalTokens)
	}
}

func TestGenerate_WithModelOption(t *testing.T) {
	srv := mockOllama(t)
	p := newTestProvider(t, srv.URL)

	resp, err := p.Generate(context.Background(), "Hello", llm.WithModel("custom-model"))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.Model != "custom-model" {
		t.Errorf("Model = %q, want %q", resp.Model, "custom-model")
	}
}

func TestGenerate_WithOptions(t *testing.T) {
	// Verify that options are sent to Ollama (the mock ignores them but
	// this tests that buildOptions doesn't cause errors).
	srv := mockOllama(t)
	p := newTestProvider(t, srv.URL)

	resp, err := p.Generate(context.Background(), "Hello",
		llm.WithTemperature(0.3),
		llm.WithMaxTokens(100),
	)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
}

func TestGenerate_Streaming(t *testing.T) {
	// The mock returns a single response chunk, but we verify the StreamFunc is called.
	srv := mockOllama(t)
	p := newTestProvider(t, srv.URL)

	var chunks []string
	resp, err := p.Generate(context.Background(), "Hello",
		llm.WithStreamFunc(func(_ context.Context, chunk []byte) error {
			chunks = append(chunks, string(chunk))
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.Content == "" {
		t.Error("expected non-empty content")
	}
	if len(chunks) == 0 {
		t.Error("StreamFunc was never called")
	}
}

func TestGenerate_CancelledContext(t *testing.T) {
	srv := mockOllama(t)
	p := newTestProvider(t, srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := p.Generate(ctx, "Hello")
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestChat_Success(t *testing.T) {
	srv := mockOllama(t)
	p := newTestProvider(t, srv.URL)

	messages := []llm.Message{
		{Role: llm.RoleSystem, Content: "You are helpful."},
		{Role: llm.RoleUser, Content: "What is 2+2?"},
	}
	resp, err := p.Chat(context.Background(), messages)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if !strings.Contains(resp.Content, "4") {
		t.Errorf("Content = %q, expected it to contain '4'", resp.Content)
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 6 {
		t.Errorf("CompletionTokens = %d, want 6", resp.Usage.CompletionTokens)
	}
}

func TestChat_EmptyMessages(t *testing.T) {
	srv := mockOllama(t)
	p := newTestProvider(t, srv.URL)

	_, err := p.Chat(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty messages")
	}
	if !llm.IsModelNotFoundError(err) && !strings.Contains(err.Error(), "empty") {
		// Verify it's an InvalidRequest error.
		var pe *llm.ProviderError
		if ok := isProviderError(err, &pe); !ok || pe.Code != llm.ErrCodeInvalidRequest {
			t.Errorf("expected ErrCodeInvalidRequest, got %v", err)
		}
	}
}

func TestHeartbeat_Success(t *testing.T) {
	srv := mockOllama(t)
	p := newTestProvider(t, srv.URL)

	if err := p.Heartbeat(context.Background()); err != nil {
		t.Errorf("Heartbeat() error = %v", err)
	}
}

func TestHeartbeat_ServerDown(t *testing.T) {
	// Point at a closed server to simulate unreachable Ollama.
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()

	p := newTestProvider(t, srv.URL)
	err := p.Heartbeat(context.Background())
	if err == nil {
		t.Fatal("expected error for closed server")
	}
}

func TestListModels_Success(t *testing.T) {
	srv := mockOllama(t)
	p := newTestProvider(t, srv.URL)

	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels() error = %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("ListModels() returned %d models, want 2", len(models))
	}
	if models[0] != "qwen2.5:32b" {
		t.Errorf("models[0] = %q, want %q", models[0], "qwen2.5:32b")
	}
	if models[1] != "llama3:8b" {
		t.Errorf("models[1] = %q, want %q", models[1], "llama3:8b")
	}
}

func TestMapError_ContextCanceled(t *testing.T) {
	err := mapError(context.Canceled)
	if !llm.IsTimeoutError(err) {
		t.Errorf("expected timeout error, got %v", err)
	}
}

func TestMapError_DeadlineExceeded(t *testing.T) {
	err := mapError(context.DeadlineExceeded)
	if !llm.IsTimeoutError(err) {
		t.Errorf("expected timeout error, got %v", err)
	}
}

func TestMapError_StatusError404Model(t *testing.T) {
	err := mapError(api.StatusError{
		StatusCode:   404,
		ErrorMessage: "model 'nonexistent' not found",
	})
	if !llm.IsModelNotFoundError(err) {
		t.Errorf("expected model-not-found error, got %v", err)
	}
}

func TestMapError_StatusError401(t *testing.T) {
	err := mapError(api.StatusError{
		StatusCode:   401,
		ErrorMessage: "unauthorized",
	})
	if !llm.IsAuthenticationError(err) {
		t.Errorf("expected authentication error, got %v", err)
	}
}

func TestMapError_StatusError500(t *testing.T) {
	err := mapError(api.StatusError{
		StatusCode:   500,
		ErrorMessage: "internal server error",
	})
	if !llm.IsServerError(err) {
		t.Errorf("expected server error, got %v", err)
	}
}

func TestMapError_Nil(t *testing.T) {
	if err := mapError(nil); err != nil {
		t.Errorf("mapError(nil) = %v, want nil", err)
	}
}

// isProviderError is a test helper that extracts a ProviderError from err.
func isProviderError(err error, pe **llm.ProviderError) bool {
	var target *llm.ProviderError
	if ok := llm.IsAuthenticationError(err); ok {
		// Not the check we want here, but errors.As will work.
	}
	// Use a simple type assertion since errors.As requires a pointer-to-pointer.
	type unwrapper interface{ Unwrap() error }
	for err != nil {
		if p, ok := err.(*llm.ProviderError); ok {
			*pe = p
			return true
		}
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
		} else {
			break
		}
	}
	_ = target
	return false
}
