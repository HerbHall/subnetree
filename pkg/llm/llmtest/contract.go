// Package llmtest provides shared contract tests that verify any
// llm.Provider implementation behaves correctly. Every provider's test
// file should call TestProviderContract to ensure conformance.
//
// These tests require a live LLM service. Use build tags or environment
// variables to skip when no service is available.
package llmtest

import (
	"context"
	"strings"
	"testing"

	"github.com/HerbHall/subnetree/pkg/llm"
)

// TestProviderContract runs a suite of behavioral contract tests against
// any llm.Provider implementation. Call this from each provider's _test.go:
//
//	func TestContract(t *testing.T) {
//	    llmtest.TestProviderContract(t, func() llm.Provider { return ollama.New(cfg) })
//	}
func TestProviderContract(t *testing.T, factory func() llm.Provider) {
	t.Helper()

	t.Run("Generate_returns_non_empty_response", func(t *testing.T) {
		p := factory()
		resp, err := p.Generate(context.Background(), "Say hello in exactly three words")
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		if resp == nil {
			t.Fatal("Generate() returned nil response")
		}
		if resp.Content == "" {
			t.Error("Generate() returned empty content")
		}
		if resp.Model == "" {
			t.Error("Response.Model must not be empty")
		}
	})

	t.Run("Chat_with_conversation_history", func(t *testing.T) {
		p := factory()
		messages := []llm.Message{
			{Role: llm.RoleSystem, Content: "You are a helpful assistant. Be concise."},
			{Role: llm.RoleUser, Content: "What is 2+2? Reply with just the number."},
		}
		resp, err := p.Chat(context.Background(), messages)
		if err != nil {
			t.Fatalf("Chat() error = %v", err)
		}
		if resp == nil {
			t.Fatal("Chat() returned nil response")
		}
		if resp.Content == "" {
			t.Error("Chat() returned empty content")
		}
		if !strings.Contains(resp.Content, "4") {
			t.Logf("Chat() response = %q", resp.Content)
			t.Error("expected response to contain '4'")
		}
	})

	t.Run("Generate_with_model_option", func(t *testing.T) {
		p := factory()
		resp, err := p.Generate(
			context.Background(),
			"Hi",
			llm.WithModel("nonexistent-model-12345"),
		)
		if err != nil {
			if llm.IsModelNotFoundError(err) {
				return // expected -- provider correctly reports missing model
			}
			// Some providers silently fall back to default model. That's OK.
			t.Logf("Generate() with bad model returned error: %v", err)
			return
		}
		if resp == nil {
			t.Fatal("Generate() returned nil response")
		}
		// Provider may have fallen back to default model.
		if resp.Model == "" {
			t.Error("Response.Model must not be empty")
		}
	})

	t.Run("Generate_cancelled_context", func(t *testing.T) {
		p := factory()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := p.Generate(ctx, "Write a very long essay about everything")
		if err == nil {
			t.Error("Generate() with cancelled context should return error")
		}
	})

	t.Run("Chat_empty_messages_returns_error", func(t *testing.T) {
		p := factory()
		_, err := p.Chat(context.Background(), nil)
		if err == nil {
			t.Error("Chat() with nil messages should return error")
		}
	})

	t.Run("HealthReporter_if_implemented", func(t *testing.T) {
		p := factory()
		hr, ok := p.(llm.HealthReporter)
		if !ok {
			t.Skip("Provider does not implement HealthReporter")
		}
		if err := hr.Heartbeat(context.Background()); err != nil {
			t.Errorf("Heartbeat() error = %v", err)
		}
		models, err := hr.ListModels(context.Background())
		if err != nil {
			t.Fatalf("ListModels() error = %v", err)
		}
		if len(models) == 0 {
			t.Error("ListModels() returned empty list")
		}
	})
}
