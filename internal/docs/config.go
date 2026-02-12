package docs

import "time"

type Config struct {
	RetentionPeriod     time.Duration `mapstructure:"retention_period"`
	MaintenanceInterval time.Duration `mapstructure:"maintenance_interval"`
	MaxSnapshotsPerApp  int           `mapstructure:"max_snapshots_per_app"`
	DockerSocket        string        `mapstructure:"docker_socket"`
	CollectInterval     time.Duration `mapstructure:"collect_interval"`
}

func DefaultConfig() Config {
	return Config{
		RetentionPeriod:     90 * 24 * time.Hour,
		MaintenanceInterval: 1 * time.Hour,
		MaxSnapshotsPerApp:  100,
		DockerSocket:        "",
		CollectInterval:     6 * time.Hour,
	}
}
