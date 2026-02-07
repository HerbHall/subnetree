package llm

import "errors"

// Error code constants for standardized error handling across providers.
// Providers map their native errors to one of these codes.
const (
	ErrCodeAuthentication = "authentication_error"
	ErrCodeRateLimit      = "rate_limit_exceeded"
	ErrCodeModelNotFound  = "model_not_found"
	ErrCodeInvalidRequest = "invalid_request"
	ErrCodeContextLength  = "context_length_exceeded"
	ErrCodeServerError    = "server_error"
	ErrCodeTimeout        = "timeout"
)

// ProviderError represents a typed error from an LLM provider.
// Use the IsXxx helpers below to classify errors without inspecting fields.
type ProviderError struct {
	Code    string // One of the ErrCode* constants.
	Message string // Human-readable description.
	Err     error  // Underlying error (may be nil).
}

func (e *ProviderError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a typed provider error.
func NewProviderError(code, message string, err error) *ProviderError {
	return &ProviderError{Code: code, Message: message, Err: err}
}

// IsAuthenticationError reports whether err is an authentication failure.
func IsAuthenticationError(err error) bool {
	return hasCode(err, ErrCodeAuthentication)
}

// IsRateLimitError reports whether err is a rate-limit error.
func IsRateLimitError(err error) bool {
	return hasCode(err, ErrCodeRateLimit)
}

// IsModelNotFoundError reports whether err is a model-not-found error.
func IsModelNotFoundError(err error) bool {
	return hasCode(err, ErrCodeModelNotFound)
}

// IsContextLengthError reports whether err is a context-length-exceeded error.
func IsContextLengthError(err error) bool {
	return hasCode(err, ErrCodeContextLength)
}

// IsServerError reports whether err is a provider-side server error.
func IsServerError(err error) bool {
	return hasCode(err, ErrCodeServerError)
}

// IsTimeoutError reports whether err is a timeout.
func IsTimeoutError(err error) bool {
	return hasCode(err, ErrCodeTimeout)
}

// IsRetryable reports whether the error is transient and the call may succeed on retry.
func IsRetryable(err error) bool {
	return IsRateLimitError(err) || IsServerError(err) || IsTimeoutError(err)
}

func hasCode(err error, code string) bool {
	var pe *ProviderError
	return errors.As(err, &pe) && pe.Code == code
}
