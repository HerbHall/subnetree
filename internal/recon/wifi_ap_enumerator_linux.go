//go:build linux

package recon

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mdlayher/wifi"
	"go.uber.org/zap"
)

type linuxAPClientEnumerator struct {
	logger *zap.Logger
}

// NewAPClientEnumerator returns a Linux APClientEnumerator backed by nl80211.
func NewAPClientEnumerator(logger *zap.Logger) APClientEnumerator {
	return &linuxAPClientEnumerator{logger: logger}
}

// Available returns true if at least one WiFi interface is in AP mode.
func (e *linuxAPClientEnumerator) Available() bool {
	c, err := wifi.New()
	if err != nil {
		e.logger.Debug("wifi client unavailable for AP enumeration", zap.Error(err))
		return false
	}
	defer c.Close()

	ifaces, err := c.Interfaces()
	if err != nil {
		e.logger.Debug("failed to enumerate wifi interfaces for AP check", zap.Error(err))
		return false
	}

	for _, ifi := range ifaces {
		if ifi.Type == wifi.InterfaceTypeAP {
			return true
		}
	}

	// Fallback: check for hostapd control sockets.
	entries, err := os.ReadDir("/var/run/hostapd")
	if err == nil && len(entries) > 0 {
		return true
	}

	return false
}

// Enumerate returns all client stations associated with AP-mode interfaces.
// Falls back to hostapd control socket parsing if nl80211 fails.
func (e *linuxAPClientEnumerator) Enumerate(ctx context.Context) ([]APClientInfo, error) {
	clients, err := e.enumerateNL80211(ctx)
	if err != nil {
		e.logger.Debug("nl80211 AP enumeration failed, trying hostapd fallback", zap.Error(err))
		return e.enumerateHostapd(ctx)
	}
	return clients, nil
}

// enumerateNL80211 uses the mdlayher/wifi library to enumerate AP clients.
func (e *linuxAPClientEnumerator) enumerateNL80211(_ context.Context) ([]APClientInfo, error) {
	c, err := wifi.New()
	if err != nil {
		return nil, fmt.Errorf("open wifi client: %w", err)
	}
	defer c.Close()

	ifaces, err := c.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("enumerate wifi interfaces: %w", err)
	}

	var clients []APClientInfo
	for _, ifi := range ifaces {
		if ifi.Type != wifi.InterfaceTypeAP {
			continue
		}

		stations, sErr := c.StationInfo(ifi)
		if sErr != nil {
			e.logger.Warn("failed to get station info for AP interface",
				zap.String("interface", ifi.Name),
				zap.Error(sErr))
			continue
		}

		apBSSID := ""
		if ifi.HardwareAddr != nil {
			apBSSID = ifi.HardwareAddr.String()
		}

		for _, sta := range stations {
			client := APClientInfo{
				Signal:        int(sta.Signal / 100), // mBm to dBm
				SignalAverage: int(sta.SignalAverage),
				Connected:     sta.Connected,
				Inactive:      sta.Inactive,
				RxBitrate:     sta.ReceiveBitrate,
				TxBitrate:     sta.TransmitBitrate,
				RxBytes:       int(sta.ReceivedBytes),
				TxBytes:       int(sta.TransmittedBytes),
				InterfaceName: ifi.Name,
				APBSSID:       apBSSID,
			}
			if sta.HardwareAddr != nil {
				client.MACAddress = sta.HardwareAddr.String()
			}
			clients = append(clients, client)
		}
	}

	return clients, nil
}

// enumerateHostapd parses the hostapd control socket to enumerate clients.
// This is a fallback for systems where nl80211 station info requires root.
func (e *linuxAPClientEnumerator) enumerateHostapd(_ context.Context) ([]APClientInfo, error) {
	entries, err := os.ReadDir("/var/run/hostapd")
	if err != nil {
		return nil, fmt.Errorf("read hostapd directory: %w", err)
	}

	var clients []APClientInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ifaceName := entry.Name()
		socketPath := "/var/run/hostapd/" + ifaceName

		ifaceClients, parseErr := e.parseHostapdSocket(socketPath, ifaceName)
		if parseErr != nil {
			e.logger.Debug("failed to parse hostapd socket",
				zap.String("socket", socketPath),
				zap.Error(parseErr))
			continue
		}
		clients = append(clients, ifaceClients...)
	}

	return clients, nil
}

// parseHostapdSocket connects to a hostapd control socket and enumerates stations.
func (e *linuxAPClientEnumerator) parseHostapdSocket(socketPath, ifaceName string) ([]APClientInfo, error) {
	// Create a temporary local socket for receiving replies.
	localPath := fmt.Sprintf("/tmp/subnetree_hostapd_%s_%d", ifaceName, time.Now().UnixNano())
	defer os.Remove(localPath)

	localAddr := &net.UnixAddr{Name: localPath, Net: "unixgram"}
	remoteAddr := &net.UnixAddr{Name: socketPath, Net: "unixgram"}

	conn, err := net.DialUnix("unixgram", localAddr, remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("connect to hostapd: %w", err)
	}
	defer conn.Close()

	// Set read timeout.
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, fmt.Errorf("set deadline: %w", err)
	}

	// Send STA-FIRST to get the first station.
	if _, err := conn.Write([]byte("STA-FIRST")); err != nil {
		return nil, fmt.Errorf("send STA-FIRST: %w", err)
	}

	var clients []APClientInfo
	buf := make([]byte, 4096)

	for {
		n, readErr := conn.Read(buf)
		if readErr != nil {
			break
		}

		response := string(buf[:n])
		if response == "" || response == "FAIL" {
			break
		}

		client := e.parseStationResponse(response, ifaceName)
		if client.MACAddress != "" {
			clients = append(clients, client)
		}

		// Request next station.
		if _, err := conn.Write([]byte("STA-NEXT " + client.MACAddress)); err != nil {
			break
		}
	}

	return clients, nil
}

// parseStationResponse parses a hostapd STA response into an APClientInfo.
func (e *linuxAPClientEnumerator) parseStationResponse(response, ifaceName string) APClientInfo {
	client := APClientInfo{
		InterfaceName: ifaceName,
	}

	scanner := bufio.NewScanner(strings.NewReader(response))
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// First line is the MAC address.
		if lineNum == 1 {
			mac := strings.TrimSpace(line)
			if _, parseErr := net.ParseMAC(mac); parseErr == nil {
				client.MACAddress = mac
			}
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "signal":
			if v, err := strconv.Atoi(val); err == nil {
				client.Signal = v
			}
		case "signal_avg":
			if v, err := strconv.Atoi(val); err == nil {
				client.SignalAverage = v
			}
		case "connected_time":
			if v, err := strconv.ParseInt(val, 10, 64); err == nil {
				client.Connected = time.Duration(v) * time.Second
			}
		case "inactive_msec":
			if v, err := strconv.ParseInt(val, 10, 64); err == nil {
				client.Inactive = time.Duration(v) * time.Millisecond
			}
		case "rx_bytes":
			if v, err := strconv.Atoi(val); err == nil {
				client.RxBytes = v
			}
		case "tx_bytes":
			if v, err := strconv.Atoi(val); err == nil {
				client.TxBytes = v
			}
		case "rx_rate_info":
			if v, err := strconv.Atoi(val); err == nil {
				client.RxBitrate = v * 100 // hostapd reports in 100kbps
			}
		case "tx_rate_info":
			if v, err := strconv.Atoi(val); err == nil {
				client.TxBitrate = v * 100
			}
		}
	}

	return client
}
