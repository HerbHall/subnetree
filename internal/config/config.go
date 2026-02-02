// Package config provides a Viper-backed implementation of the plugin.Config interface.
package config

import (
	"time"

	"github.com/HerbHall/netvantage/pkg/plugin"
	"github.com/spf13/viper"
)

// Compile-time interface guard.
var _ plugin.Config = (*ViperConfig)(nil)

// ViperConfig wraps a Viper instance to implement plugin.Config.
type ViperConfig struct {
	v *viper.Viper
}

// New creates a Config backed by the given Viper instance.
// Returns the concrete type; callers assign to plugin.Config where needed.
func New(v *viper.Viper) *ViperConfig {
	if v == nil {
		v = viper.New()
	}
	return &ViperConfig{v: v}
}

func (c *ViperConfig) Unmarshal(target any) error {
	return c.v.Unmarshal(target)
}

func (c *ViperConfig) Get(key string) any {
	return c.v.Get(key)
}

func (c *ViperConfig) GetString(key string) string {
	return c.v.GetString(key)
}

func (c *ViperConfig) GetInt(key string) int {
	return c.v.GetInt(key)
}

func (c *ViperConfig) GetBool(key string) bool {
	return c.v.GetBool(key)
}

func (c *ViperConfig) GetDuration(key string) time.Duration {
	return c.v.GetDuration(key)
}

func (c *ViperConfig) IsSet(key string) bool {
	return c.v.IsSet(key)
}

func (c *ViperConfig) Sub(key string) plugin.Config {
	sub := c.v.Sub(key)
	if sub == nil {
		return New(nil)
	}
	return New(sub)
}

// Viper returns the underlying Viper instance for direct access
// (e.g., by the server for top-level config like server.port).
func (c *ViperConfig) Viper() *viper.Viper {
	return c.v
}
