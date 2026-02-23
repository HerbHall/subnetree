package recon

import "context"

// WifiScanner discovers nearby WiFi access points via OS APIs.
type WifiScanner interface {
	Available() bool
	Scan(ctx context.Context) ([]AccessPointInfo, error)
}

// AccessPointInfo represents a discovered WiFi access point.
type AccessPointInfo struct {
	BSSID     string // MAC address of the AP
	SSID      string // Network name
	Channel   int    // WiFi channel number
	Frequency int    // MHz (2412=ch1, 5180=ch36, etc.)
	Signal    int    // dBm (typically -30 to -90)
	Security  string // "WPA2", "WPA3", "WPA2/WPA3", "WEP", "Open"
}

// freqToChannel converts a WiFi center frequency in MHz to a channel number.
// Returns 0 for unrecognised frequencies.
func freqToChannel(freqMHz int) int {
	switch {
	// 2.4 GHz band: channels 1-14
	case freqMHz >= 2412 && freqMHz <= 2484:
		if freqMHz == 2484 {
			return 14 // Japan channel 14
		}
		return (freqMHz - 2412) / 5 + 1

	// 5 GHz band: channels 36-177
	case freqMHz >= 5180 && freqMHz <= 5885:
		return (freqMHz - 5180) / 5 + 36

	// 6 GHz band (WiFi 6E): channels 1-233
	case freqMHz >= 5955 && freqMHz <= 7115:
		return (freqMHz - 5955) / 5 + 1
	}
	return 0
}

// qualityToDBm converts a Windows WLAN signal quality percentage (0-100) to an
// approximate dBm value.  The formula mirrors the inverse of the Windows NDIS
// mapping: quality = 2 * (dBm + 100), clamped to [0, 100].
func qualityToDBm(quality int) int {
	if quality <= 0 {
		return -100
	}
	if quality >= 100 {
		return -50
	}
	return (quality / 2) - 100
}

// authAlgoToSecurity maps a Windows DOT11_AUTH_ALGORITHM value to a
// human-readable security string.
//
// Reference: https://learn.microsoft.com/en-us/windows/win32/nativewifi/dot11-auth-algorithm
func authAlgoToSecurity(algo int) string {
	switch algo {
	case 1: // DOT11_AUTH_ALGO_80211_OPEN
		return "Open"
	case 2: // DOT11_AUTH_ALGO_80211_SHARED_KEY
		return "WEP"
	case 3: // DOT11_AUTH_ALGO_WPA
		return "WPA"
	case 4: // DOT11_AUTH_ALGO_WPA_PSK
		return "WPA"
	case 5: // DOT11_AUTH_ALGO_WPA_NONE
		return "Open"
	case 6: // DOT11_AUTH_ALGO_RSNA (WPA2-Enterprise)
		return "WPA2"
	case 7: // DOT11_AUTH_ALGO_RSNA_PSK (WPA2-Personal)
		return "WPA2"
	case 8: // DOT11_AUTH_ALGO_WPA3 (WPA3-Enterprise 192-bit)
		return "WPA3"
	case 9: // DOT11_AUTH_ALGO_WPA3_SAE (WPA3-Personal)
		return "WPA3"
	case 10: // DOT11_AUTH_ALGO_OWE
		return "OWE"
	case 11: // DOT11_AUTH_ALGO_WPA3_ENT
		return "WPA3"
	default:
		return "Unknown"
	}
}
