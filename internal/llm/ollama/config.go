package ollama

import "time"

// Config holds the Ollama provider configuration.
type Config struct {
	URL     string        `mapstructure:"url"`
	Model   string        `mapstructure:"model"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// DefaultConfig returns sensible defaults for local Ollama.
func DefaultConfig() Config {
	return Config{
		URL:     "http://localhost:11434",
		Model:   "qwen2.5:32b",
		Timeout: 5 * time.Minute,
	}
}
