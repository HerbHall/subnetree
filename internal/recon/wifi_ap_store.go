package recon

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// WiFiClientSnapshot represents a WiFi client's signal and traffic data.
type WiFiClientSnapshot struct {
	DeviceID     string    `json:"device_id"`
	SignalDBm    int       `json:"signal_dbm"`
	SignalAvgDBm int       `json:"signal_avg_dbm"`
	ConnectedSec int64     `json:"connected_sec"`
	InactiveSec  int64     `json:"inactive_sec"`
	RxBitrate    int       `json:"rx_bitrate_bps"`
	TxBitrate    int       `json:"tx_bitrate_bps"`
	RxBytes      int       `json:"rx_bytes"`
	TxBytes      int       `json:"tx_bytes"`
	APBSSID      string    `json:"ap_bssid"`
	APSSID       string    `json:"ap_ssid"`
	CollectedAt  time.Time `json:"collected_at"`
}

// UpsertWiFiClient inserts or replaces a WiFi client snapshot.
func (s *ReconStore) UpsertWiFiClient(ctx context.Context, snap *WiFiClientSnapshot) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO recon_wifi_clients (
		device_id, signal_dbm, signal_avg_dbm,
		connected_sec, inactive_sec,
		rx_bitrate_bps, tx_bitrate_bps,
		rx_bytes, tx_bytes,
		ap_bssid, ap_ssid, collected_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snap.DeviceID, snap.SignalDBm, snap.SignalAvgDBm,
		snap.ConnectedSec, snap.InactiveSec,
		snap.RxBitrate, snap.TxBitrate,
		snap.RxBytes, snap.TxBytes,
		snap.APBSSID, snap.APSSID, snap.CollectedAt)
	if err != nil {
		return fmt.Errorf("upsert wifi client: %w", err)
	}
	return nil
}

// GetWiFiClient returns the WiFi client snapshot for a specific device.
// Returns nil, nil if not found.
func (s *ReconStore) GetWiFiClient(ctx context.Context, deviceID string) (*WiFiClientSnapshot, error) {
	var snap WiFiClientSnapshot
	err := s.db.QueryRowContext(ctx, `SELECT
		device_id, signal_dbm, signal_avg_dbm,
		connected_sec, inactive_sec,
		rx_bitrate_bps, tx_bitrate_bps,
		rx_bytes, tx_bytes,
		ap_bssid, ap_ssid, collected_at
		FROM recon_wifi_clients WHERE device_id = ?`, deviceID).Scan(
		&snap.DeviceID, &snap.SignalDBm, &snap.SignalAvgDBm,
		&snap.ConnectedSec, &snap.InactiveSec,
		&snap.RxBitrate, &snap.TxBitrate,
		&snap.RxBytes, &snap.TxBytes,
		&snap.APBSSID, &snap.APSSID, &snap.CollectedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get wifi client: %w", err)
	}
	return &snap, nil
}

// ListWiFiClients returns WiFi client snapshots. If apDeviceID is non-empty,
// only clients whose parent device matches the AP are returned.
func (s *ReconStore) ListWiFiClients(ctx context.Context, apDeviceID string) ([]WiFiClientSnapshot, error) {
	query := `SELECT
		wc.device_id, wc.signal_dbm, wc.signal_avg_dbm,
		wc.connected_sec, wc.inactive_sec,
		wc.rx_bitrate_bps, wc.tx_bitrate_bps,
		wc.rx_bytes, wc.tx_bytes,
		wc.ap_bssid, wc.ap_ssid, wc.collected_at
		FROM recon_wifi_clients wc`

	var args []any
	if apDeviceID != "" {
		query += ` JOIN recon_devices d ON d.id = wc.device_id
			WHERE d.parent_device_id = ?`
		args = append(args, apDeviceID)
	}
	query += " ORDER BY wc.device_id"

	rows, err := s.db.QueryContext(ctx, query, args...) //nolint:gosec // query uses parameterized placeholders only
	if err != nil {
		return nil, fmt.Errorf("list wifi clients: %w", err)
	}
	defer rows.Close()

	var result []WiFiClientSnapshot
	for rows.Next() {
		var snap WiFiClientSnapshot
		if err := rows.Scan(
			&snap.DeviceID, &snap.SignalDBm, &snap.SignalAvgDBm,
			&snap.ConnectedSec, &snap.InactiveSec,
			&snap.RxBitrate, &snap.TxBitrate,
			&snap.RxBytes, &snap.TxBytes,
			&snap.APBSSID, &snap.APSSID, &snap.CollectedAt,
		); err != nil {
			return nil, fmt.Errorf("scan wifi client row: %w", err)
		}
		result = append(result, snap)
	}
	return result, rows.Err()
}
