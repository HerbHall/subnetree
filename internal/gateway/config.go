package gateway

import "time"

// GatewayConfig holds configuration for the Gateway module.
type GatewayConfig struct {
	SessionTimeout      time.Duration `mapstructure:"session_timeout"`
	MaxSessions         int           `mapstructure:"max_sessions"`
	AuditRetentionDays  int           `mapstructure:"audit_retention_days"`
	MaintenanceInterval time.Duration `mapstructure:"maintenance_interval"`
	DefaultProxyPort    int           `mapstructure:"default_proxy_port"`
}

// DefaultConfig returns the default Gateway configuration.
func DefaultConfig() GatewayConfig {
	return GatewayConfig{
		SessionTimeout:      30 * time.Minute,
		MaxSessions:         100,
		AuditRetentionDays:  90,
		MaintenanceInterval: 5 * time.Minute,
		DefaultProxyPort:    80,
	}
}
