//go:build windows

package recon

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

type windowsAPClientEnumerator struct {
	logger *zap.Logger
}

// NewAPClientEnumerator returns a Windows APClientEnumerator that uses netsh
// and ARP table parsing for Mobile Hotspot / Hosted Network enumeration.
func NewAPClientEnumerator(logger *zap.Logger) APClientEnumerator {
	return &windowsAPClientEnumerator{logger: logger}
}

// Available returns true if a Windows hosted network or mobile hotspot is active.
func (e *windowsAPClientEnumerator) Available() bool {
	out, err := exec.Command("netsh", "wlan", "show", "hostednetwork").Output()
	if err != nil {
		e.logger.Debug("netsh hostednetwork check failed", zap.Error(err))
		return false
	}
	return strings.Contains(string(out), "Status") &&
		strings.Contains(string(out), "Started")
}

// Enumerate returns clients connected to the Windows hotspot by cross-referencing
// the hosted network config with the ARP table for the hotspot subnet.
func (e *windowsAPClientEnumerator) Enumerate(ctx context.Context) ([]APClientInfo, error) {
	// Parse hosted network info for BSSID and SSID.
	apBSSID, apSSID, parseErr := e.parseHostedNetwork(ctx)
	if parseErr != nil {
		return nil, fmt.Errorf("parse hosted network: %w", parseErr)
	}

	// Windows ICS (Internet Connection Sharing) typically assigns the
	// hotspot interface 192.168.137.1 with subnet 192.168.137.0/24.
	hotspotPrefix := "192.168.137."

	// Read ARP table and filter to hotspot subnet.
	arpOut, err := exec.CommandContext(ctx, "arp", "-a").Output()
	if err != nil {
		return nil, fmt.Errorf("read ARP table: %w", err)
	}

	var clients []APClientInfo
	scanner := bufio.NewScanner(strings.NewReader(string(arpOut)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		ip := fields[0]
		mac := fields[1]

		// Only include entries from the hotspot subnet.
		if !strings.HasPrefix(ip, hotspotPrefix) {
			continue
		}

		// Skip the gateway itself and broadcast entries.
		if ip == "192.168.137.1" || ip == "192.168.137.255" {
			continue
		}

		// Normalise MAC format from Windows (xx-xx-xx-xx-xx-xx) to standard.
		mac = strings.ReplaceAll(mac, "-", ":")
		if mac == "ff:ff:ff:ff:ff:ff" {
			continue
		}

		clients = append(clients, APClientInfo{
			MACAddress: mac,
			APBSSID:    apBSSID,
			APSSID:     apSSID,
		})
	}

	return clients, nil
}

// parseHostedNetwork extracts the BSSID and SSID from netsh output.
func (e *windowsAPClientEnumerator) parseHostedNetwork(ctx context.Context) (bssid, ssid string, err error) {
	out, cmdErr := exec.CommandContext(ctx, "netsh", "wlan", "show", "hostednetwork").Output()
	if cmdErr != nil {
		return "", "", fmt.Errorf("netsh: %w", cmdErr)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "BSSID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				bssid = strings.TrimSpace(parts[1])
				bssid = strings.ReplaceAll(bssid, "-", ":")
			}
		}
		if strings.HasPrefix(line, "SSID name") || strings.HasPrefix(line, "SSID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				ssid = strings.Trim(strings.TrimSpace(parts[1]), "\"")
			}
		}
	}

	return bssid, ssid, nil
}
