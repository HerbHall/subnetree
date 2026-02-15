package openai

import "time"

// Config holds the OpenAI provider configuration.
type Config struct {
	Model        string        `mapstructure:"model"`
	Timeout      time.Duration `mapstructure:"timeout"`
	CredentialID string        `mapstructure:"credential_id"`
}

// DefaultConfig returns sensible defaults for OpenAI.
func DefaultConfig() Config {
	return Config{
		Model:   "gpt-4o-mini",
		Timeout: 2 * time.Minute,
	}
}
