package scout

import "path/filepath"

// Config holds the Scout agent configuration.
type Config struct {
	ServerAddr    string `mapstructure:"server_addr"`
	CheckInterval int    `mapstructure:"check_interval_seconds"`
	AgentID       string `mapstructure:"agent_id"`
	EnrollToken   string `mapstructure:"enroll_token"`
	CertPath      string `mapstructure:"cert_path"`
	KeyPath       string `mapstructure:"key_path"`
	CACertPath    string `mapstructure:"ca_cert_path"` // path to CA certificate for TLS verification
	Insecure      bool   `mapstructure:"insecure"`     // skip TLS (dev/testing only)
}

// DefaultConfig returns the default agent configuration.
func DefaultConfig() *Config {
	return &Config{
		ServerAddr:    "localhost:9090",
		CheckInterval: 30,
		Insecure:      true, // backward compat: insecure by default until TLS is configured
	}
}

// ResolveCACertPath returns the CA certificate path, defaulting to the
// same directory as CertPath with filename "ca.crt" when CACertPath is empty.
func (c *Config) ResolveCACertPath() string {
	if c.CACertPath != "" {
		return c.CACertPath
	}
	if c.CertPath != "" {
		return filepath.Join(filepath.Dir(c.CertPath), "ca.crt")
	}
	return ""
}
