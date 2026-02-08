package server

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds the server configuration.
type Config struct {
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	DataDir string `mapstructure:"data_dir"`
}

// Addr returns the listen address as host:port.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// LoadConfig reads configuration from file and environment variables.
func LoadConfig(configPath string) (*viper.Viper, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.data_dir", "./data")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.dsn", "./data/subnetree.db")

	// Plugin defaults
	v.SetDefault("plugins.recon.enabled", true)
	v.SetDefault("plugins.recon.scan_timeout", "5m")
	v.SetDefault("plugins.recon.ping_timeout", "2s")
	v.SetDefault("plugins.recon.ping_count", 3)
	v.SetDefault("plugins.recon.concurrency", 64)
	v.SetDefault("plugins.recon.arp_enabled", true)
	v.SetDefault("plugins.recon.device_lost_after", "24h")
	v.SetDefault("plugins.pulse.enabled", true)
	v.SetDefault("plugins.pulse.check_interval", "30s")
	v.SetDefault("plugins.pulse.ping_timeout", "5s")
	v.SetDefault("plugins.pulse.ping_count", 3)
	v.SetDefault("plugins.pulse.consecutive_failures", 3)
	v.SetDefault("plugins.pulse.retention_period", "720h")
	v.SetDefault("plugins.pulse.max_workers", 10)
	v.SetDefault("plugins.pulse.maintenance_interval", "1h")
	v.SetDefault("plugins.dispatch.enabled", true)
	v.SetDefault("plugins.vault.enabled", true)
	v.SetDefault("plugins.vault.audit_retention_period", "2160h")
	v.SetDefault("plugins.vault.maintenance_interval", "1h")
	v.SetDefault("plugins.gateway.enabled", true)
	v.SetDefault("plugins.gateway.session_timeout", "30m")
	v.SetDefault("plugins.gateway.max_sessions", 100)
	v.SetDefault("plugins.gateway.audit_retention_days", 90)
	v.SetDefault("plugins.gateway.maintenance_interval", "5m")
	v.SetDefault("plugins.gateway.default_proxy_port", 80)
	v.SetDefault("plugins.webhook.enabled", true)
	v.SetDefault("plugins.webhook.url", "")
	v.SetDefault("plugins.webhook.timeout", "10s")
	v.SetDefault("plugins.llm.url", "http://localhost:11434")
	v.SetDefault("plugins.llm.model", "qwen2.5:32b")
	v.SetDefault("plugins.llm.timeout", "5m")
	v.SetDefault("plugins.insight.ewma_alpha", 0.1)
	v.SetDefault("plugins.insight.learning_period", "168h")
	v.SetDefault("plugins.insight.min_samples_stable", 168)
	v.SetDefault("plugins.insight.zscore_threshold", 3.0)
	v.SetDefault("plugins.insight.cusum_drift", 0.5)
	v.SetDefault("plugins.insight.cusum_threshold", 5.0)
	v.SetDefault("plugins.insight.forecast_window", "168h")
	v.SetDefault("plugins.insight.anomaly_retention", "720h")
	v.SetDefault("plugins.insight.maintenance_interval", "1h")

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("subnetree")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/subnetree")
	}

	// Environment variable support: NV_SERVER_PORT=9090
	v.SetEnvPrefix("NV")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
		// Config file not found is fine -- use defaults
	}

	return v, nil
}
