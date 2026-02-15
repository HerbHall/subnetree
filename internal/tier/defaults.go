package tier

import (
	"github.com/HerbHall/subnetree/pkg/catalog"
	"github.com/spf13/viper"
)

// TierDefaults maps hardware tiers to configuration overrides.
// These are applied ONLY for keys not already set by user config.
var TierDefaults = map[catalog.HardwareTier]map[string]any{
	catalog.TierSBC: {
		"plugins.pulse.check_interval": "5m",
		"plugins.pulse.max_workers":    2,
		"plugins.recon.scan_interval":  "5m",
		"data_retention_days":          7,
		"plugins.insight.enabled":      false,
		"plugins.llm.enabled":          false,
	},
	catalog.TierMiniPC: {
		"plugins.pulse.check_interval": "2m",
		"plugins.pulse.max_workers":    5,
		"plugins.recon.scan_interval":  "2m",
		"data_retention_days":          30,
	},
	catalog.TierCluster: {
		"plugins.pulse.check_interval": "1m",
		"plugins.pulse.max_workers":    10,
		"plugins.recon.scan_interval":  "1m",
		"data_retention_days":          90,
	},
	catalog.TierSMB: {
		"plugins.pulse.check_interval": "1m",
		"plugins.pulse.max_workers":    10,
		"plugins.recon.scan_interval":  "1m",
		"data_retention_days":          180,
	},
}

// ApplyDefaults sets tier-specific defaults for keys not already configured by the user.
func ApplyDefaults(v *viper.Viper, t catalog.HardwareTier) {
	defaults, ok := TierDefaults[t]
	if !ok {
		return
	}
	for key, val := range defaults {
		if !v.IsSet(key) {
			v.SetDefault(key, val)
		}
	}
}
