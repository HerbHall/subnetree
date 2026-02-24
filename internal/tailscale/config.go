package tailscale

import "time"

// TailscaleConfig holds configuration for the Tailscale integration plugin.
type TailscaleConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	CredentialID string        `mapstructure:"credential_id"` //nolint:gosec // G101: config field name, not a credential
	Tailnet      string        `mapstructure:"tailnet"`
	SyncInterval time.Duration `mapstructure:"sync_interval"`
	BaseURL      string        `mapstructure:"base_url"`
}

// DefaultConfig returns sensible defaults for the Tailscale plugin.
func DefaultConfig() TailscaleConfig {
	return TailscaleConfig{
		Tailnet:      "-",
		SyncInterval: 5 * time.Minute,
		BaseURL:      "https://api.tailscale.com",
	}
}
