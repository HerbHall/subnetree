package tier

import (
	"testing"

	"github.com/HerbHall/subnetree/pkg/catalog"
	"github.com/spf13/viper"
)

func TestDetectTierWithRAM(t *testing.T) {
	tests := []struct {
		name     string
		ramBytes uint64
		arch     string
		cores    int
		want     catalog.HardwareTier
	}{
		{
			name:     "ARM 2GB -> TierSBC",
			ramBytes: 2 * gb,
			arch:     "arm64",
			cores:    4,
			want:     catalog.TierSBC,
		},
		{
			name:     "x86 4GB -> TierSBC",
			ramBytes: 4 * gb,
			arch:     "amd64",
			cores:    4,
			want:     catalog.TierSBC,
		},
		{
			name:     "x86 16GB -> TierMiniPC",
			ramBytes: 16 * gb,
			arch:     "amd64",
			cores:    4,
			want:     catalog.TierMiniPC,
		},
		{
			name:     "x86 64GB 8 cores -> TierCluster",
			ramBytes: 64 * gb,
			arch:     "amd64",
			cores:    8,
			want:     catalog.TierCluster,
		},
		{
			name:     "x86 128GB -> TierSMB",
			ramBytes: 128 * gb,
			arch:     "amd64",
			cores:    32,
			want:     catalog.TierSMB,
		},
		{
			name:     "zero RAM -> TierSBC",
			ramBytes: 0,
			arch:     "amd64",
			cores:    4,
			want:     catalog.TierSBC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectTierWithRAM(tt.ramBytes, tt.arch, tt.cores)
			if got != tt.want {
				t.Errorf("DetectTierWithRAM(%d, %q, %d) = %d, want %d", tt.ramBytes, tt.arch, tt.cores, got, tt.want)
			}
		})
	}
}

func TestName(t *testing.T) {
	tests := []struct {
		tier catalog.HardwareTier
		want string
	}{
		{catalog.TierSBC, "SBC"},
		{catalog.TierMiniPC, "Mini PC"},
		{catalog.TierNAS, "NAS"},
		{catalog.TierCluster, "Cluster"},
		{catalog.TierSMB, "SMB Server"},
		{catalog.HardwareTier(99), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := Name(tt.tier); got != tt.want {
				t.Errorf("Name(%d) = %q, want %q", tt.tier, got, tt.want)
			}
		})
	}
}

func TestApplyDefaults_DoesNotOverrideUserConfig(t *testing.T) {
	v := viper.New()
	// User explicitly set a value.
	v.Set("plugins.pulse.check_interval", "10s")
	v.Set("plugins.pulse.max_workers", 20)

	ApplyDefaults(v, catalog.TierSBC)

	// User values must be preserved.
	if got := v.GetString("plugins.pulse.check_interval"); got != "10s" {
		t.Errorf("check_interval = %q, want %q (user override should be preserved)", got, "10s")
	}
	if got := v.GetInt("plugins.pulse.max_workers"); got != 20 {
		t.Errorf("max_workers = %d, want %d (user override should be preserved)", got, 20)
	}

	// Tier defaults applied for unset keys.
	if got := v.GetInt("data_retention_days"); got != 7 {
		t.Errorf("data_retention_days = %d, want %d (tier default should apply)", got, 7)
	}
}

func TestApplyDefaults_SetsDefaultsForUnsetKeys(t *testing.T) {
	v := viper.New()
	ApplyDefaults(v, catalog.TierMiniPC)

	if got := v.GetString("plugins.pulse.check_interval"); got != "2m" {
		t.Errorf("check_interval = %q, want %q", got, "2m")
	}
	if got := v.GetInt("plugins.pulse.max_workers"); got != 5 {
		t.Errorf("max_workers = %d, want %d", got, 5)
	}
	if got := v.GetInt("data_retention_days"); got != 30 {
		t.Errorf("data_retention_days = %d, want %d", got, 30)
	}
}

func TestApplyDefaults_UnknownTierIsNoOp(t *testing.T) {
	v := viper.New()
	ApplyDefaults(v, catalog.TierNAS) // NAS has no defaults

	if v.IsSet("plugins.pulse.check_interval") {
		t.Error("expected no defaults set for TierNAS")
	}
}
