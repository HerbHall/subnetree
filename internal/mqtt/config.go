package mqtt

import "time"

// Config holds MQTT publisher configuration.
type Config struct {
	BrokerURL   string        `mapstructure:"broker_url"`
	Username    string        `mapstructure:"username"`
	Password    string        `mapstructure:"password"` //nolint:gosec // G101: config field name, not a credential
	ClientID    string        `mapstructure:"client_id"`
	TopicPrefix string        `mapstructure:"topic_prefix"`
	QoS         byte          `mapstructure:"qos"`
	Retain      bool          `mapstructure:"retain"`
	UseTLS      bool          `mapstructure:"use_tls"`
	Timeout     time.Duration `mapstructure:"timeout"`

	// Home Assistant MQTT auto-discovery settings.
	HADiscovery       bool   `mapstructure:"ha_discovery"`        // Enable HA auto-discovery (default: false)
	HADiscoveryPrefix string `mapstructure:"ha_discovery_prefix"` // HA discovery topic prefix (default: "homeassistant")
}

// DefaultConfig returns sensible defaults for the MQTT publisher.
func DefaultConfig() Config {
	return Config{
		BrokerURL:         "", // disabled by default
		ClientID:          "subnetree",
		TopicPrefix:       "subnetree",
		QoS:               1,
		Retain:            false,
		Timeout:           10 * time.Second,
		HADiscovery:       false,
		HADiscoveryPrefix: "homeassistant",
	}
}
