package tier

import (
	"runtime"

	"github.com/HerbHall/subnetree/pkg/catalog"
)

const (
	gb = 1024 * 1024 * 1024 // bytes in a gigabyte
)

// DetectTier determines the hardware tier based on system resources.
func DetectTier() catalog.HardwareTier {
	totalRAM := getSystemRAMBytes()
	arch := runtime.GOARCH
	cores := runtime.NumCPU()

	return DetectTierWithRAM(totalRAM, arch, cores)
}

// Name returns a human-readable name for a hardware tier.
func Name(t catalog.HardwareTier) string {
	switch t {
	case catalog.TierSBC:
		return "SBC"
	case catalog.TierMiniPC:
		return "Mini PC"
	case catalog.TierNAS:
		return "NAS"
	case catalog.TierCluster:
		return "Cluster"
	case catalog.TierSMB:
		return "SMB Server"
	default:
		return "Unknown"
	}
}

// DetectTierWithRAM is exported for testing -- allows injecting RAM value.
func DetectTierWithRAM(ramBytes uint64, arch string, cores int) catalog.HardwareTier {
	isARM := arch == "arm" || arch == "arm64"

	switch {
	case isARM && ramBytes < 8*gb:
		return catalog.TierSBC
	case ramBytes < 8*gb:
		return catalog.TierSBC
	case ramBytes <= 32*gb:
		return catalog.TierMiniPC
	case ramBytes <= 64*gb && cores <= 16:
		return catalog.TierCluster
	default:
		return catalog.TierSMB
	}
}
