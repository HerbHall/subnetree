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
	v.SetDefault("plugins.dispatch.enabled", true)
	v.SetDefault("plugins.vault.enabled", true)
	v.SetDefault("plugins.gateway.enabled", true)
	v.SetDefault("plugins.webhook.enabled", true)
	v.SetDefault("plugins.webhook.url", "")
	v.SetDefault("plugins.webhook.timeout", "10s")
	v.SetDefault("plugins.llm.url", "http://localhost:11434")
	v.SetDefault("plugins.llm.model", "qwen2.5:32b")
	v.SetDefault("plugins.llm.timeout", "5m")

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
