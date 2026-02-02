package recon

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HerbHall/netvantage/pkg/models"
	"github.com/google/uuid"
)

// ReconStore provides database operations for the Recon module.
type ReconStore struct {
	db *sql.DB
}

// NewReconStore creates a new ReconStore backed by the given database.
func NewReconStore(db *sql.DB) *ReconStore {
	return &ReconStore{db: db}
}

// TopologyLink represents a network connection between two devices.
type TopologyLink struct {
	ID             string    `json:"id"`
	SourceDeviceID string    `json:"source_device_id"`
	TargetDeviceID string    `json:"target_device_id"`
	SourcePort     string    `json:"source_port"`
	TargetPort     string    `json:"target_port"`
	LinkType       string    `json:"link_type"`
	Speed          int       `json:"speed"`
	DiscoveredAt   time.Time `json:"discovered_at"`
	LastConfirmed  time.Time `json:"last_confirmed"`
}

// ListDevicesOptions controls pagination and filtering for device queries.
type ListDevicesOptions struct {
	Limit  int
	Offset int
	Status string
	ScanID string
}

// UpsertDevice inserts a new device or updates an existing one.
// If the device has a MAC address, it matches by MAC; otherwise by IP.
// Returns the final device record and whether it was newly created.
func (s *ReconStore) UpsertDevice(ctx context.Context, device *models.Device) (created bool, err error) {
	now := time.Now().UTC()

	// Try to find existing device by MAC first, then by first IP.
	var existing *models.Device
	if device.MACAddress != "" {
		existing, _ = s.GetDeviceByMAC(ctx, device.MACAddress)
	}
	if existing == nil && len(device.IPAddresses) > 0 {
		existing, _ = s.GetDeviceByIP(ctx, device.IPAddresses[0])
	}

	if existing != nil {
		// Merge IP addresses.
		ipSet := make(map[string]bool)
		for _, ip := range existing.IPAddresses {
			ipSet[ip] = true
		}
		for _, ip := range device.IPAddresses {
			ipSet[ip] = true
		}
		merged := make([]string, 0, len(ipSet))
		for ip := range ipSet {
			merged = append(merged, ip)
		}

		ipsJSON, _ := json.Marshal(merged)
		mac := existing.MACAddress
		if device.MACAddress != "" {
			mac = device.MACAddress
		}
		manufacturer := existing.Manufacturer
		if device.Manufacturer != "" {
			manufacturer = device.Manufacturer
		}
		method := existing.DiscoveryMethod
		if device.DiscoveryMethod != "" {
			method = device.DiscoveryMethod
		}

		_, err = s.db.ExecContext(ctx, `
			UPDATE recon_devices SET
				ip_addresses = ?, mac_address = ?, manufacturer = ?,
				status = ?, discovery_method = ?, last_seen = ?
			WHERE id = ?`,
			string(ipsJSON), mac, manufacturer,
			string(models.DeviceStatusOnline), string(method), now,
			existing.ID,
		)
		if err != nil {
			return false, fmt.Errorf("update device: %w", err)
		}
		device.ID = existing.ID
		return false, nil
	}

	// Create new device.
	if device.ID == "" {
		device.ID = uuid.New().String()
	}
	ipsJSON, _ := json.Marshal(device.IPAddresses)
	tagsJSON, _ := json.Marshal(device.Tags)
	if device.Tags == nil {
		tagsJSON = []byte("[]")
	}
	cfJSON, _ := json.Marshal(device.CustomFields)
	if device.CustomFields == nil {
		cfJSON = []byte("{}")
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO recon_devices (
			id, hostname, ip_addresses, mac_address, manufacturer,
			device_type, os, status, discovery_method, agent_id,
			first_seen, last_seen, notes, tags, custom_fields
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		device.ID, device.Hostname, string(ipsJSON), device.MACAddress, device.Manufacturer,
		string(device.DeviceType), device.OS, string(device.Status), string(device.DiscoveryMethod), device.AgentID,
		now, now, device.Notes, string(tagsJSON), string(cfJSON),
	)
	if err != nil {
		return false, fmt.Errorf("insert device: %w", err)
	}
	device.FirstSeen = now
	device.LastSeen = now
	return true, nil
}

// GetDevice returns a device by ID.
func (s *ReconStore) GetDevice(ctx context.Context, id string) (*models.Device, error) {
	return s.scanDevice(s.db.QueryRowContext(ctx, `SELECT
		id, hostname, ip_addresses, mac_address, manufacturer,
		device_type, os, status, discovery_method, agent_id,
		first_seen, last_seen, notes, tags, custom_fields
		FROM recon_devices WHERE id = ?`, id))
}

// GetDeviceByMAC returns a device by MAC address.
func (s *ReconStore) GetDeviceByMAC(ctx context.Context, mac string) (*models.Device, error) {
	return s.scanDevice(s.db.QueryRowContext(ctx, `SELECT
		id, hostname, ip_addresses, mac_address, manufacturer,
		device_type, os, status, discovery_method, agent_id,
		first_seen, last_seen, notes, tags, custom_fields
		FROM recon_devices WHERE mac_address = ?`, mac))
}

// GetDeviceByIP returns the first device matching the given IP address.
func (s *ReconStore) GetDeviceByIP(ctx context.Context, ip string) (*models.Device, error) {
	// SQLite JSON: check if ip_addresses array contains the IP.
	return s.scanDevice(s.db.QueryRowContext(ctx, `SELECT
		id, hostname, ip_addresses, mac_address, manufacturer,
		device_type, os, status, discovery_method, agent_id,
		first_seen, last_seen, notes, tags, custom_fields
		FROM recon_devices WHERE ip_addresses LIKE ?`, "%\""+ip+"\"%"))
}

// ListDevices returns a paginated list of devices.
func (s *ReconStore) ListDevices(ctx context.Context, opts ListDevicesOptions) ([]models.Device, int, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	// Build WHERE clause.
	where := "1=1"
	args := []any{}
	if opts.Status != "" {
		where += " AND status = ?"
		args = append(args, opts.Status)
	}
	if opts.ScanID != "" {
		where += " AND id IN (SELECT device_id FROM recon_scan_devices WHERE scan_id = ?)"
		args = append(args, opts.ScanID)
	}

	// Count total.
	// The where clause is built above using only ? placeholders; no user input is concatenated.
	var total int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM recon_devices WHERE "+where, args..., //nolint:gosec // where uses parameterized placeholders only
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count devices: %w", err)
	}

	// Query with pagination.
	queryArgs := make([]any, 0, len(args)+2)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, opts.Limit, opts.Offset)
	//nolint:gosec // where uses parameterized placeholders only
	rows, err := s.db.QueryContext(ctx, "SELECT "+
		"id, hostname, ip_addresses, mac_address, manufacturer, "+
		"device_type, os, status, discovery_method, agent_id, "+
		"first_seen, last_seen, notes, tags, custom_fields "+
		"FROM recon_devices WHERE "+where+" ORDER BY last_seen DESC LIMIT ? OFFSET ?",
		queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		d, err := s.scanDeviceRow(rows)
		if err != nil {
			return nil, 0, err
		}
		devices = append(devices, *d)
	}
	return devices, total, rows.Err()
}

// CreateScan inserts a new scan record.
func (s *ReconStore) CreateScan(ctx context.Context, scan *models.ScanResult) error {
	if scan.ID == "" {
		scan.ID = uuid.New().String()
	}
	if scan.StartedAt == "" {
		scan.StartedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if scan.Status == "" {
		scan.Status = "running"
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO recon_scans (id, subnet, started_at, status)
		VALUES (?, ?, ?, ?)`,
		scan.ID, scan.Subnet, scan.StartedAt, scan.Status,
	)
	if err != nil {
		return fmt.Errorf("insert scan: %w", err)
	}
	return nil
}

// UpdateScan updates a scan's status and counts.
func (s *ReconStore) UpdateScan(ctx context.Context, scan *models.ScanResult) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE recon_scans SET status = ?, ended_at = ?, total = ?, online = ?, error_msg = ?
		WHERE id = ?`,
		scan.Status, scan.EndedAt, scan.Total, scan.Online, "", scan.ID,
	)
	if err != nil {
		return fmt.Errorf("update scan: %w", err)
	}
	return nil
}

// UpdateScanError updates a scan with failure status and error message.
func (s *ReconStore) UpdateScanError(ctx context.Context, scanID, errMsg string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE recon_scans SET status = 'failed', ended_at = ?, error_msg = ?
		WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), errMsg, scanID,
	)
	return err
}

// GetScan returns a scan by ID.
func (s *ReconStore) GetScan(ctx context.Context, id string) (*models.ScanResult, error) {
	var scan models.ScanResult
	var endedAt sql.NullString
	var errorMsg string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, subnet, started_at, ended_at, status, total, online, error_msg
		FROM recon_scans WHERE id = ?`, id,
	).Scan(&scan.ID, &scan.Subnet, &scan.StartedAt, &endedAt, &scan.Status, &scan.Total, &scan.Online, &errorMsg)
	if err != nil {
		return nil, fmt.Errorf("get scan: %w", err)
	}
	if endedAt.Valid {
		scan.EndedAt = endedAt.String
	}
	return &scan, nil
}

// ListScans returns a paginated list of scans.
func (s *ReconStore) ListScans(ctx context.Context, limit, offset int) ([]models.ScanResult, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, subnet, started_at, ended_at, status, total, online, error_msg
		FROM recon_scans ORDER BY started_at DESC LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list scans: %w", err)
	}
	defer rows.Close()

	var scans []models.ScanResult
	for rows.Next() {
		var scan models.ScanResult
		var endedAt sql.NullString
		var errorMsg string
		if err := rows.Scan(&scan.ID, &scan.Subnet, &scan.StartedAt, &endedAt, &scan.Status, &scan.Total, &scan.Online, &errorMsg); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		if endedAt.Valid {
			scan.EndedAt = endedAt.String
		}
		scans = append(scans, scan)
	}
	return scans, rows.Err()
}

// LinkScanDevice associates a device with a scan.
func (s *ReconStore) LinkScanDevice(ctx context.Context, scanID, deviceID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO recon_scan_devices (scan_id, device_id) VALUES (?, ?)`,
		scanID, deviceID,
	)
	return err
}

// UpsertTopologyLink creates or updates a topology link between two devices.
func (s *ReconStore) UpsertTopologyLink(ctx context.Context, link *TopologyLink) error {
	if link.ID == "" {
		link.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO recon_topology_links (
			id, source_device_id, target_device_id, source_port, target_port,
			link_type, speed, discovered_at, last_confirmed
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (source_device_id, target_device_id, link_type)
		DO UPDATE SET last_confirmed = ?`,
		link.ID, link.SourceDeviceID, link.TargetDeviceID, link.SourcePort, link.TargetPort,
		link.LinkType, link.Speed, now, now,
		now,
	)
	if err != nil {
		return fmt.Errorf("upsert topology link: %w", err)
	}
	return nil
}

// GetTopologyLinks returns all topology links.
func (s *ReconStore) GetTopologyLinks(ctx context.Context) ([]TopologyLink, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source_device_id, target_device_id, source_port, target_port,
			link_type, speed, discovered_at, last_confirmed
		FROM recon_topology_links`)
	if err != nil {
		return nil, fmt.Errorf("get topology links: %w", err)
	}
	defer rows.Close()

	var links []TopologyLink
	for rows.Next() {
		var l TopologyLink
		if err := rows.Scan(&l.ID, &l.SourceDeviceID, &l.TargetDeviceID,
			&l.SourcePort, &l.TargetPort, &l.LinkType, &l.Speed,
			&l.DiscoveredAt, &l.LastConfirmed); err != nil {
			return nil, fmt.Errorf("scan topology row: %w", err)
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// scanDevice scans a single row into a Device.
func (s *ReconStore) scanDevice(row *sql.Row) (*models.Device, error) {
	var d models.Device
	var ipsJSON, tagsJSON, cfJSON string
	var dt, status, method string
	err := row.Scan(
		&d.ID, &d.Hostname, &ipsJSON, &d.MACAddress, &d.Manufacturer,
		&dt, &d.OS, &status, &method, &d.AgentID,
		&d.FirstSeen, &d.LastSeen, &d.Notes, &tagsJSON, &cfJSON,
	)
	if err != nil {
		return nil, err
	}
	d.DeviceType = models.DeviceType(dt)
	d.Status = models.DeviceStatus(status)
	d.DiscoveryMethod = models.DiscoveryMethod(method)
	_ = json.Unmarshal([]byte(ipsJSON), &d.IPAddresses)
	_ = json.Unmarshal([]byte(tagsJSON), &d.Tags)
	_ = json.Unmarshal([]byte(cfJSON), &d.CustomFields)
	return &d, nil
}

// scanDeviceRow scans a *sql.Rows row into a Device.
func (s *ReconStore) scanDeviceRow(rows *sql.Rows) (*models.Device, error) {
	var d models.Device
	var ipsJSON, tagsJSON, cfJSON string
	var dt, status, method string
	err := rows.Scan(
		&d.ID, &d.Hostname, &ipsJSON, &d.MACAddress, &d.Manufacturer,
		&dt, &d.OS, &status, &method, &d.AgentID,
		&d.FirstSeen, &d.LastSeen, &d.Notes, &tagsJSON, &cfJSON,
	)
	if err != nil {
		return nil, err
	}
	d.DeviceType = models.DeviceType(dt)
	d.Status = models.DeviceStatus(status)
	d.DiscoveryMethod = models.DiscoveryMethod(method)
	_ = json.Unmarshal([]byte(ipsJSON), &d.IPAddresses)
	_ = json.Unmarshal([]byte(tagsJSON), &d.Tags)
	_ = json.Unmarshal([]byte(cfJSON), &d.CustomFields)
	return &d, nil
}
