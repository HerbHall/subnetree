package recon

import (
	"context"
	"time"
)

// APClientEnumerator enumerates clients associated with the server's own
// WiFi access point or hotspot interface.
type APClientEnumerator interface {
	Available() bool
	Enumerate(ctx context.Context) ([]APClientInfo, error)
}

// APClientInfo represents a client station associated with the server's AP.
type APClientInfo struct {
	MACAddress    string        // Client MAC address
	Signal        int           // dBm (negative, e.g., -65)
	SignalAverage int           // dBm average
	Connected     time.Duration // Time since association
	Inactive      time.Duration // Time since last activity
	RxBitrate     int           // bits/sec
	TxBitrate     int           // bits/sec
	RxBytes       int
	TxBytes       int
	InterfaceName string // AP interface name (e.g., "wlan0")
	APBSSID       string // MAC address of the AP interface
	APSSID        string // SSID of the AP network
}
