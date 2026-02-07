package ollama

import (
	"context"
	"errors"
	"strings"

	"github.com/HerbHall/subnetree/pkg/llm"
	"github.com/ollama/ollama/api"
)

// mapError translates Ollama and network errors into typed llm.ProviderError values.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	// Context errors.
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return llm.NewProviderError(llm.ErrCodeTimeout, "request timed out or cancelled", err)
	}

	// Ollama StatusError (HTTP-level errors).
	var se api.StatusError
	if errors.As(err, &se) {
		switch {
		case se.StatusCode == 401:
			return llm.NewProviderError(llm.ErrCodeAuthentication, se.ErrorMessage, err)
		case se.StatusCode == 404 && strings.Contains(strings.ToLower(se.ErrorMessage), "model"):
			return llm.NewProviderError(llm.ErrCodeModelNotFound, se.ErrorMessage, err)
		case se.StatusCode >= 500:
			return llm.NewProviderError(llm.ErrCodeServerError, se.ErrorMessage, err)
		case se.StatusCode >= 400:
			return llm.NewProviderError(llm.ErrCodeInvalidRequest, se.ErrorMessage, err)
		}
	}

	// Connection refused, DNS errors, etc.
	msg := err.Error()
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "dial tcp") {
		return llm.NewProviderError(llm.ErrCodeServerError, "ollama server unreachable", err)
	}

	return llm.NewProviderError(llm.ErrCodeServerError, "ollama error", err)
}
