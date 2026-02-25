package netbox

import "time"

// Config holds the NetBox integration configuration.
type Config struct {
	URL      string        `mapstructure:"url"`       // NetBox base URL (e.g., "https://netbox.example.com")
	Token    string        `mapstructure:"token"`     // API token
	SiteID   int           `mapstructure:"site_id"`   // Default site ID for new devices (0 = auto-create)
	SiteName string        `mapstructure:"site_name"` // Default site name (used when SiteID=0)
	TagName  string        `mapstructure:"tag_name"`  // Tag for SubNetree-managed devices (default: "subnetree-managed")
	DryRun   bool          `mapstructure:"dry_run"`   // Default dry-run mode
	Timeout  time.Duration `mapstructure:"timeout"`   // HTTP client timeout (default: 30s)
}

// DefaultConfig returns a Config with sensible defaults.
// URL is empty, meaning the module is disabled until configured.
func DefaultConfig() Config {
	return Config{
		TagName: "subnetree-managed",
		Timeout: 30 * time.Second,
	}
}
