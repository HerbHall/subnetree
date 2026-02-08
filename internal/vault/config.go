package vault

import "time"

// VaultConfig holds configuration for the Vault module.
type VaultConfig struct {
	AuditRetentionPeriod time.Duration `mapstructure:"audit_retention_period"`
	MaintenanceInterval  time.Duration `mapstructure:"maintenance_interval"`
}

// DefaultConfig returns the default Vault configuration.
func DefaultConfig() VaultConfig {
	return VaultConfig{
		AuditRetentionPeriod: 90 * 24 * time.Hour, // 90 days
		MaintenanceInterval:  1 * time.Hour,
	}
}
