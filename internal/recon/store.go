package recon

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
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
	Limit      int
	Offset     int
	Status     string
	DeviceType string
	ScanID     string
}

// UpdateDeviceParams holds partial update fields for a device.
type UpdateDeviceParams struct {
	Notes        *string            `json:"notes,omitempty"`
	Tags         *[]string          `json:"tags,omitempty"`
	CustomFields *map[string]string `json:"custom_fields,omitempty"`
	DeviceType   *string            `json:"device_type,omitempty"`
}

// DeviceStatusChange records a status transition for a device.
type DeviceStatusChange struct {
	ID        string    `json:"id"`
	DeviceID  string    `json:"device_id"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
	ChangedAt time.Time `json:"changed_at"`
}

// ScanSummary is a lightweight scan record for device scan history.
type ScanSummary struct {
	ID        string `json:"id"`
	Subnet    string `json:"subnet"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
	Total     int    `json:"total"`
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

		newStatus := string(models.DeviceStatusOnline)

		_, err = s.db.ExecContext(ctx, `
			UPDATE recon_devices SET
				ip_addresses = ?, mac_address = ?, manufacturer = ?,
				status = ?, discovery_method = ?, last_seen = ?
			WHERE id = ?`,
			string(ipsJSON), mac, manufacturer,
			newStatus, string(method), now,
			existing.ID,
		)
		if err != nil {
			return false, fmt.Errorf("update device: %w", err)
		}

		// Record status change if status actually changed.
		oldStatus := string(existing.Status)
		if oldStatus != newStatus {
			s.recordStatusChange(ctx, existing.ID, oldStatus, newStatus)
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
	if opts.DeviceType != "" {
		where += " AND device_type = ?"
		args = append(args, opts.DeviceType)
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

// FindStaleDevices returns devices that are currently online but haven't been
// seen since before the given threshold time.
func (s *ReconStore) FindStaleDevices(ctx context.Context, threshold time.Time) ([]models.Device, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT
		id, hostname, ip_addresses, mac_address, manufacturer,
		device_type, os, status, discovery_method, agent_id,
		first_seen, last_seen, notes, tags, custom_fields
		FROM recon_devices WHERE status = ? AND last_seen < ?`,
		string(models.DeviceStatusOnline), threshold,
	)
	if err != nil {
		return nil, fmt.Errorf("find stale devices: %w", err)
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		d, err := s.scanDeviceRow(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *d)
	}
	return devices, rows.Err()
}

// MarkDeviceOffline sets a device's status to offline and records the change.
func (s *ReconStore) MarkDeviceOffline(ctx context.Context, deviceID string) error {
	// Read current status before updating.
	var oldStatus string
	err := s.db.QueryRowContext(ctx,
		`SELECT status FROM recon_devices WHERE id = ?`, deviceID,
	).Scan(&oldStatus)
	if err != nil {
		return fmt.Errorf("read device status: %w", err)
	}

	newStatus := string(models.DeviceStatusOffline)
	_, err = s.db.ExecContext(ctx,
		`UPDATE recon_devices SET status = ? WHERE id = ?`,
		newStatus, deviceID,
	)
	if err != nil {
		return fmt.Errorf("mark device offline: %w", err)
	}

	if oldStatus != newStatus {
		s.recordStatusChange(ctx, deviceID, oldStatus, newStatus)
	}
	return nil
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

// recordStatusChange inserts a row into recon_device_history.
// Errors are silently ignored so callers are not disrupted by history failures.
func (s *ReconStore) recordStatusChange(ctx context.Context, deviceID, oldStatus, newStatus string) {
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO recon_device_history (id, device_id, old_status, new_status, changed_at)
		VALUES (?, ?, ?, ?, ?)`,
		uuid.New().String(), deviceID, oldStatus, newStatus, time.Now().UTC(),
	)
}

// UpdateDevice applies a partial update to an existing device.
func (s *ReconStore) UpdateDevice(ctx context.Context, id string, params UpdateDeviceParams) error {
	// Verify the device exists.
	existing, err := s.GetDevice(ctx, id)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	if params.Notes != nil {
		_, err = s.db.ExecContext(ctx, `UPDATE recon_devices SET notes = ? WHERE id = ?`, *params.Notes, id)
		if err != nil {
			return fmt.Errorf("update notes: %w", err)
		}
	}
	if params.Tags != nil {
		tagsJSON, _ := json.Marshal(*params.Tags)
		_, err = s.db.ExecContext(ctx, `UPDATE recon_devices SET tags = ? WHERE id = ?`, string(tagsJSON), id)
		if err != nil {
			return fmt.Errorf("update tags: %w", err)
		}
	}
	if params.CustomFields != nil {
		cfJSON, _ := json.Marshal(*params.CustomFields)
		_, err = s.db.ExecContext(ctx, `UPDATE recon_devices SET custom_fields = ? WHERE id = ?`, string(cfJSON), id)
		if err != nil {
			return fmt.Errorf("update custom_fields: %w", err)
		}
	}
	if params.DeviceType != nil {
		oldType := string(existing.DeviceType)
		if *params.DeviceType != oldType {
			_, err = s.db.ExecContext(ctx, `UPDATE recon_devices SET device_type = ? WHERE id = ?`, *params.DeviceType, id)
			if err != nil {
				return fmt.Errorf("update device_type: %w", err)
			}
		}
	}
	return nil
}

// DeleteDevice removes a device by ID. Returns an error if the device does not exist.
func (s *ReconStore) DeleteDevice(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM recon_devices WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// InsertManualDevice creates a new device record with discovery_method=manual.
func (s *ReconStore) InsertManualDevice(ctx context.Context, device *models.Device) error {
	now := time.Now().UTC()
	if device.ID == "" {
		device.ID = uuid.New().String()
	}
	if device.Status == "" {
		device.Status = models.DeviceStatusUnknown
	}
	device.DiscoveryMethod = models.DiscoveryManual
	device.FirstSeen = now
	device.LastSeen = now

	ipsJSON, _ := json.Marshal(device.IPAddresses)
	if device.IPAddresses == nil {
		ipsJSON = []byte("[]")
	}
	tagsJSON, _ := json.Marshal(device.Tags)
	if device.Tags == nil {
		tagsJSON = []byte("[]")
	}
	cfJSON, _ := json.Marshal(device.CustomFields)
	if device.CustomFields == nil {
		cfJSON = []byte("{}")
	}

	_, err := s.db.ExecContext(ctx, `
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
		return fmt.Errorf("insert manual device: %w", err)
	}
	return nil
}

// GetDeviceHistory returns paginated status change history for a device.
func (s *ReconStore) GetDeviceHistory(ctx context.Context, deviceID string, limit, offset int) ([]DeviceStatusChange, int, error) {
	if limit <= 0 {
		limit = 50
	}

	var total int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM recon_device_history WHERE device_id = ?`, deviceID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count device history: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, device_id, old_status, new_status, changed_at
		FROM recon_device_history
		WHERE device_id = ?
		ORDER BY changed_at DESC
		LIMIT ? OFFSET ?`,
		deviceID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list device history: %w", err)
	}
	defer rows.Close()

	var changes []DeviceStatusChange
	for rows.Next() {
		var c DeviceStatusChange
		if err := rows.Scan(&c.ID, &c.DeviceID, &c.OldStatus, &c.NewStatus, &c.ChangedAt); err != nil {
			return nil, 0, fmt.Errorf("scan history row: %w", err)
		}
		changes = append(changes, c)
	}
	return changes, total, rows.Err()
}

// GetDeviceScans returns scans that discovered or updated a given device.
func (s *ReconStore) GetDeviceScans(ctx context.Context, deviceID string, limit, offset int) ([]ScanSummary, int, error) {
	if limit <= 0 {
		limit = 50
	}

	var total int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM recon_scan_devices sd
		 JOIN recon_scans sc ON sc.id = sd.scan_id
		 WHERE sd.device_id = ?`, deviceID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count device scans: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT sc.id, sc.subnet, sc.status, sc.started_at, sc.total
		FROM recon_scan_devices sd
		JOIN recon_scans sc ON sc.id = sd.scan_id
		WHERE sd.device_id = ?
		ORDER BY sc.started_at DESC
		LIMIT ? OFFSET ?`,
		deviceID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list device scans: %w", err)
	}
	defer rows.Close()

	var scans []ScanSummary
	for rows.Next() {
		var sc ScanSummary
		if err := rows.Scan(&sc.ID, &sc.Subnet, &sc.Status, &sc.StartedAt, &sc.Total); err != nil {
			return nil, 0, fmt.Errorf("scan summary row: %w", err)
		}
		scans = append(scans, sc)
	}
	return scans, total, rows.Err()
}
