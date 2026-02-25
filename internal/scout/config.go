package scout

import (
	"path/filepath"
	"time"
)

// Config holds the Scout agent configuration.
type Config struct {
	ServerAddr    string `mapstructure:"server_addr"`
	CheckInterval int    `mapstructure:"check_interval_seconds"`
	AgentID       string `mapstructure:"agent_id"`
	EnrollToken   string `mapstructure:"enroll_token"`
	CertPath      string `mapstructure:"cert_path"`
	KeyPath       string `mapstructure:"key_path"`
	CACertPath       string        `mapstructure:"ca_cert_path"`       // path to CA certificate for TLS verification
	Insecure         bool          `mapstructure:"insecure"`           // skip TLS (dev/testing only)
	RenewalThreshold time.Duration `mapstructure:"renewal_threshold"` // renew when cert expires within this (default 30 days)
	AutoRestart      bool          `mapstructure:"auto_restart"`      // enable init-system-aware restart on version rejection
	AutoUpdate       bool          `mapstructure:"auto_update"`       // enable automatic binary self-update
}

// DefaultConfig returns the default agent configuration.
func DefaultConfig() *Config {
	return &Config{
		ServerAddr:       "localhost:9090",
		CheckInterval:    30,
		Insecure:         true,              // backward compat: insecure by default until TLS is configured
		RenewalThreshold: 30 * 24 * time.Hour, // renew when cert expires within 30 days
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
