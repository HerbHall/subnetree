package anthropic

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/HerbHall/subnetree/pkg/llm"
)

// anthropicStatusError represents an HTTP error response from the Anthropic API.
type anthropicStatusError struct {
	StatusCode int
	Type       string
	Message    string
}

func (e *anthropicStatusError) Error() string {
	return fmt.Sprintf("anthropic: %d %s: %s", e.StatusCode, e.Type, e.Message)
}

// mapError translates Anthropic and network errors into typed llm.ProviderError values.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return llm.NewProviderError(llm.ErrCodeTimeout, "request timed out or cancelled", err)
	}

	var se *anthropicStatusError
	if errors.As(err, &se) {
		switch {
		case se.StatusCode == 401:
			return llm.NewProviderError(llm.ErrCodeAuthentication, se.Message, err)
		case se.StatusCode == 429:
			return llm.NewProviderError(llm.ErrCodeRateLimit, se.Message, err)
		case se.Type == "not_found_error":
			return llm.NewProviderError(llm.ErrCodeModelNotFound, se.Message, err)
		case se.Type == "invalid_request_error" &&
			(strings.Contains(strings.ToLower(se.Message), "token") ||
				strings.Contains(strings.ToLower(se.Message), "context")):
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
		return llm.NewProviderError(llm.ErrCodeServerError, "anthropic server unreachable", err)
	}

	return llm.NewProviderError(llm.ErrCodeServerError, "anthropic error", err)
}
