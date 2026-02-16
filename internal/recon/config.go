package recon

import "time"

// ReconConfig holds the Recon module configuration.
type ReconConfig struct {
	ScanTimeout     time.Duration  `mapstructure:"scan_timeout"`
	PingTimeout     time.Duration  `mapstructure:"ping_timeout"`
	PingCount       int            `mapstructure:"ping_count"`
	Concurrency     int            `mapstructure:"concurrency"`
	ARPEnabled      bool           `mapstructure:"arp_enabled"`
	DeviceLostAfter time.Duration  `mapstructure:"device_lost_after"`
	MDNSEnabled     bool           `mapstructure:"mdns_enabled"`
	MDNSInterval    time.Duration  `mapstructure:"mdns_interval"`
	UPNPEnabled     bool           `mapstructure:"upnp_enabled"`
	UPNPInterval    time.Duration  `mapstructure:"upnp_interval"`
	Schedule        ScheduleConfig `mapstructure:"schedule"`
}

// ScheduleConfig holds configuration for recurring scheduled scans.
type ScheduleConfig struct {
	Enabled    bool          `mapstructure:"enabled"`
	Interval   time.Duration `mapstructure:"interval"`
	QuietStart string        `mapstructure:"quiet_start"`
	QuietEnd   string        `mapstructure:"quiet_end"`
	Subnet     string        `mapstructure:"subnet"`
}

// DefaultConfig returns the default configuration for the Recon module.
func DefaultConfig() ReconConfig {
	return ReconConfig{
		ScanTimeout:     5 * time.Minute,
		PingTimeout:     2 * time.Second,
		PingCount:       3,
		Concurrency:     64,
		ARPEnabled:      true,
		DeviceLostAfter: 24 * time.Hour,
		MDNSEnabled:     true,
		MDNSInterval:    60 * time.Second,
		UPNPEnabled:     true,
		UPNPInterval:    5 * time.Minute,
		Schedule: ScheduleConfig{
			Enabled:  false,
			Interval: time.Hour,
		},
	}
}
