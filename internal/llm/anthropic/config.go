package anthropic

import "time"

// Config holds the Anthropic provider configuration.
type Config struct {
	Model        string        `mapstructure:"model"`
	Timeout      time.Duration `mapstructure:"timeout"`
	CredentialID string        `mapstructure:"credential_id"`
}

// DefaultConfig returns sensible defaults for Anthropic.
func DefaultConfig() Config {
	return Config{
		Model:   "claude-sonnet-4-5-20250929",
		Timeout: 2 * time.Minute,
	}
}
