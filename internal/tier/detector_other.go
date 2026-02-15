//go:build !linux && !windows

package tier

func getSystemRAMBytes() uint64 {
	return 0 // Unknown platform; defaults to TierSBC (conservative)
}
