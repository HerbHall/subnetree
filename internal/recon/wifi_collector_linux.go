//go:build linux

package recon

import (
	"context"
	"errors"
	"fmt"

	"github.com/mdlayher/wifi"
	"go.uber.org/zap"
)

type linuxWifiScanner struct {
	logger *zap.Logger
}

// NewWifiScanner returns a Linux WifiScanner backed by nl80211.
func NewWifiScanner(logger *zap.Logger) WifiScanner {
	return &linuxWifiScanner{logger: logger}
}

// Available returns true if at least one WiFi interface exists on this system.
func (s *linuxWifiScanner) Available() bool {
	c, err := wifi.New()
	if err != nil {
		s.logger.Debug("wifi client unavailable", zap.Error(err))
		return false
	}
	defer c.Close()

	ifaces, err := c.Interfaces()
	if err != nil {
		s.logger.Debug("failed to enumerate wifi interfaces", zap.Error(err))
		return false
	}

	for _, ifi := range ifaces {
		if ifi.Type == wifi.InterfaceTypeStation || ifi.Type == wifi.InterfaceTypeAP {
			return true
		}
	}
	return false
}

// Scan discovers nearby WiFi access points via nl80211.
// Requires root or CAP_NET_ADMIN; returns an empty slice on permission errors.
func (s *linuxWifiScanner) Scan(ctx context.Context) ([]AccessPointInfo, error) {
	c, err := wifi.New()
	if err != nil {
		if isPermissionError(err) {
			s.logger.Warn("wifi scan requires root or CAP_NET_ADMIN, skipping")
			return nil, nil
		}
		return nil, fmt.Errorf("open wifi client: %w", err)
	}
	defer c.Close()

	ifaces, err := c.Interfaces()
	if err != nil {
		if isPermissionError(err) {
			s.logger.Warn("wifi interface enumeration requires elevated privileges, skipping")
			return nil, nil
		}
		return nil, fmt.Errorf("enumerate wifi interfaces: %w", err)
	}

	// Find the first station-mode interface for scanning.
	var ifi *wifi.Interface
	for _, candidate := range ifaces {
		if candidate.Type == wifi.InterfaceTypeStation {
			ifi = candidate
			break
		}
	}
	if ifi == nil {
		s.logger.Debug("no station-mode wifi interface found")
		return nil, nil
	}

	// Trigger an active scan. This is best-effort; if the kernel rejects it
	// (e.g. already scanning, or no permission) we fall back to cached results.
	if scanErr := c.Scan(ctx, ifi); scanErr != nil {
		if isPermissionError(scanErr) {
			s.logger.Warn("wifi active scan requires elevated privileges, using cached results")
		} else if !errors.Is(scanErr, wifi.ErrScanAborted) {
			s.logger.Debug("wifi active scan failed, using cached results", zap.Error(scanErr))
		}
	}

	bssList, err := c.AccessPoints(ifi)
	if err != nil {
		if isPermissionError(err) {
			s.logger.Warn("wifi BSS list requires elevated privileges, skipping")
			return nil, nil
		}
		return nil, fmt.Errorf("get access points: %w", err)
	}

	results := make([]AccessPointInfo, 0, len(bssList))
	for _, bss := range bssList {
		if bss.BSSID == nil {
			continue
		}

		ap := AccessPointInfo{
			BSSID:     bss.BSSID.String(),
			SSID:      bss.SSID,
			Frequency: bss.Frequency,
			Channel:   freqToChannel(bss.Frequency),
			Signal:    int(bss.Signal / 100), // mBm to dBm
			Security:  rsnToSecurity(bss.RSN),
		}
		results = append(results, ap)
	}

	return results, nil
}

// rsnToSecurity converts RSN information to a human-readable security string.
func rsnToSecurity(rsn wifi.RSNInfo) string {
	if !rsn.IsInitialized() {
		return "Open"
	}

	hasWPA3 := false
	hasWPA2 := false
	for _, akm := range rsn.AKMs {
		switch akm {
		case wifi.RSNAkmSAE, wifi.RSNAkmFTSAE:
			hasWPA3 = true
		case wifi.RSNAkmPSK, wifi.RSNAkmFTPSK, wifi.RSNAkm8021X, wifi.RSNAkmFT8021X:
			hasWPA2 = true
		}
	}

	switch {
	case hasWPA3 && hasWPA2:
		return "WPA2/WPA3"
	case hasWPA3:
		return "WPA3"
	case hasWPA2:
		return "WPA2"
	}

	// Fall back to cipher analysis for older networks.
	for _, cipher := range rsn.PairwiseCiphers {
		switch cipher {
		case wifi.RSNCipherCCMP128, wifi.RSNCipherGCMP128, wifi.RSNCipherGCMP256, wifi.RSNCipherCCMP256:
			return "WPA2"
		case wifi.RSNCipherTKIP:
			return "WPA"
		case wifi.RSNCipherWEP40, wifi.RSNCipherWEP104:
			return "WEP"
		}
	}

	return "Unknown"
}

// isPermissionError checks whether the error is a permission-related error.
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "permission denied") || contains(msg, "operation not permitted")
}

// contains is a simple substring check to avoid importing strings in this file.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
