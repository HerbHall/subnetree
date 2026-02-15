package openai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/HerbHall/subnetree/pkg/llm"
)

// openaiStatusError represents an HTTP error response from the OpenAI API.
type openaiStatusError struct {
	StatusCode int
	Type       string
	Message    string
}

func (e *openaiStatusError) Error() string {
	return fmt.Sprintf("openai: %d %s: %s", e.StatusCode, e.Type, e.Message)
}

// mapError translates OpenAI and network errors into typed llm.ProviderError values.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return llm.NewProviderError(llm.ErrCodeTimeout, "request timed out or cancelled", err)
	}

	var se *openaiStatusError
	if errors.As(err, &se) {
		switch {
		case se.StatusCode == 401:
			return llm.NewProviderError(llm.ErrCodeAuthentication, se.Message, err)
		case se.StatusCode == 429:
			return llm.NewProviderError(llm.ErrCodeRateLimit, se.Message, err)
		case se.StatusCode == 404 && strings.Contains(strings.ToLower(se.Message), "model"):
			return llm.NewProviderError(llm.ErrCodeModelNotFound, se.Message, err)
		case se.Type == "context_length_exceeded" ||
			strings.Contains(strings.ToLower(se.Message), "context length"):
			return llm.NewProviderError(llm.ErrCodeContextLength, se.Message, err)
		case se.StatusCode >= 500:
			return llm.NewProviderError(llm.ErrCodeServerError, se.Message, err)
		case se.StatusCode >= 400:
			return llm.NewProviderError(llm.ErrCodeInvalidRequest, se.Message, err)
		}
	}

	msg := err.Error()
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "dial tcp") {
		return llm.NewProviderError(llm.ErrCodeServerError, "openai server unreachable", err)
	}

	return llm.NewProviderError(llm.ErrCodeServerError, "openai error", err)
}
