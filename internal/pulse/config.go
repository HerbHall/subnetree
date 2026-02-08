package pulse

import "time"

type PulseConfig struct {
	CheckInterval       time.Duration `mapstructure:"check_interval"`
	PingTimeout         time.Duration `mapstructure:"ping_timeout"`
	PingCount           int           `mapstructure:"ping_count"`
	ConsecutiveFailures int           `mapstructure:"consecutive_failures"`
	RetentionPeriod     time.Duration `mapstructure:"retention_period"`
	MaxWorkers          int           `mapstructure:"max_workers"`
	MaintenanceInterval time.Duration `mapstructure:"maintenance_interval"`
}

func DefaultConfig() PulseConfig {
	return PulseConfig{
		CheckInterval:       30 * time.Second,
		PingTimeout:         5 * time.Second,
		PingCount:           3,
		ConsecutiveFailures: 3,
		RetentionPeriod:     30 * 24 * time.Hour,
		MaxWorkers:          10,
		MaintenanceInterval: 1 * time.Hour,
	}
}
