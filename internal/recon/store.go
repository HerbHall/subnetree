package recon

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

// TopologyLayout represents a saved topology layout configuration.
type TopologyLayout struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Positions string `json:"positions"` // JSON array of {id, x, y}
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
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
	Category   string
	Owner      string
}

// UpdateDeviceParams holds partial update fields for a device.
type UpdateDeviceParams struct {
	Notes        *string            `json:"notes,omitempty"`
	Tags         *[]string          `json:"tags,omitempty"`
	CustomFields *map[string]string `json:"custom_fields,omitempty"`
	DeviceType   *string            `json:"device_type,omitempty"`
	Location     *string            `json:"location,omitempty"`
	Category     *string            `json:"category,omitempty"`
	PrimaryRole  *string            `json:"primary_role,omitempty"`
	Owner        *string            `json:"owner,omitempty"`
}

// InventorySummary provides aggregate statistics about the device inventory.
type InventorySummary struct {
	TotalDevices int            `json:"total_devices"`
	OnlineCount  int            `json:"online_count"`
	OfflineCount int            `json:"offline_count"`
	StaleCount   int            `json:"stale_count"`
	ByCategory   map[string]int `json:"by_category"`
	ByType       map[string]int `json:"by_type"`
}

// DeviceStatusChange records a status transition for a device.
type DeviceStatusChange struct {
	ID        string    `json:"id"`
	DeviceID  string    `json:"device_id"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
	ChangedAt time.Time `json:"changed_at"`
}

// ScanMetricsAggregate represents a time-bucketed aggregate of scan metrics.
type ScanMetricsAggregate struct {
	ID              string  `json:"id"`
	Period          string  `json:"period"`
	PeriodStart     string  `json:"period_start"`
	PeriodEnd       string  `json:"period_end"`
	ScanCount       int     `json:"scan_count"`
	AvgDurationMs   float64 `json:"avg_duration_ms"`
	AvgPingPhaseMs  float64 `json:"avg_ping_phase_ms"`
	AvgEnrichMs     float64 `json:"avg_enrich_ms"`
	AvgDevicesFound float64 `json:"avg_devices_found"`
	MaxDevicesFound int     `json:"max_devices_found"`
	MinDevicesFound int     `json:"min_devices_found"`
	AvgHostsAlive   float64 `json:"avg_hosts_alive"`
	TotalNewDevices int     `json:"total_new_devices"`
	FailedScans     int     `json:"failed_scans"`
	CreatedAt       string  `json:"created_at"`
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
		hostname := existing.Hostname
		if device.Hostname != "" {
			hostname = device.Hostname
		}
		osField := existing.OS
		if device.OS != "" {
			osField = device.OS
		}
		location := existing.Location
		if device.Location != "" {
			location = device.Location
		}
		category := existing.Category
		if device.Category != "" {
			category = device.Category
		}
		primaryRole := existing.PrimaryRole
		if device.PrimaryRole != "" {
			primaryRole = device.PrimaryRole
		}
		owner := existing.Owner
		if device.Owner != "" {
			owner = device.Owner
		}
		// Merge tags: combine both sets.
		mergedTags := existing.Tags
		if len(device.Tags) > 0 {
			tagSet := make(map[string]bool)
			for _, t := range existing.Tags {
				tagSet[t] = true
			}
			for _, t := range device.Tags {
				tagSet[t] = true
			}
			mergedTags = make([]string, 0, len(tagSet))
			for t := range tagSet {
				mergedTags = append(mergedTags, t)
			}
		}
		tagsJSON, _ := json.Marshal(mergedTags)
		if mergedTags == nil {
			tagsJSON = []byte("[]")
		}
		method := existing.DiscoveryMethod
		if device.DiscoveryMethod != "" {
			method = device.DiscoveryMethod
		}
		deviceType := string(existing.DeviceType)
		if existing.DeviceType == models.DeviceTypeUnknown && device.DeviceType != models.DeviceTypeUnknown {
			deviceType = string(device.DeviceType)
		}

		newStatus := string(models.DeviceStatusOnline)

		// Merge classification metadata: keep higher confidence.
		classConfidence := device.ClassificationConfidence
		classSource := device.ClassificationSource
		classSignals := device.ClassificationSignals
		if existing.ClassificationConfidence > classConfidence {
			classConfidence = existing.ClassificationConfidence
			classSource = existing.ClassificationSource
			classSignals = existing.ClassificationSignals
		}

		_, err = s.db.ExecContext(ctx, `
			UPDATE recon_devices SET
				ip_addresses = ?, mac_address = ?, manufacturer = ?,
				hostname = ?, os = ?, location = ?, category = ?, primary_role = ?, owner = ?, tags = ?,
				status = ?, discovery_method = ?, device_type = ?, last_seen = ?,
				classification_confidence = ?, classification_source = ?, classification_signals = ?
			WHERE id = ?`,
			string(ipsJSON), mac, manufacturer,
			hostname, osField, location, category, primaryRole, owner, string(tagsJSON),
			newStatus, string(method), deviceType, now,
			classConfidence, classSource, classSignals,
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
			first_seen, last_seen, notes, tags, custom_fields,
			location, category, primary_role, owner,
			classification_confidence, classification_source, classification_signals,
			parent_device_id, network_layer
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		device.ID, device.Hostname, string(ipsJSON), device.MACAddress, device.Manufacturer,
		string(device.DeviceType), device.OS, string(device.Status), string(device.DiscoveryMethod), device.AgentID,
		now, now, device.Notes, string(tagsJSON), string(cfJSON),
		device.Location, device.Category, device.PrimaryRole, device.Owner,
		device.ClassificationConfidence, device.ClassificationSource, device.ClassificationSignals,
		device.ParentDeviceID, device.NetworkLayer,
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
		first_seen, last_seen, notes, tags, custom_fields,
		location, category, primary_role, owner,
		classification_confidence, classification_source, classification_signals,
		parent_device_id, network_layer
		FROM recon_devices WHERE id = ?`, id))
}

// GetDeviceByMAC returns a device by MAC address.
func (s *ReconStore) GetDeviceByMAC(ctx context.Context, mac string) (*models.Device, error) {
	return s.scanDevice(s.db.QueryRowContext(ctx, `SELECT
		id, hostname, ip_addresses, mac_address, manufacturer,
		device_type, os, status, discovery_method, agent_id,
		first_seen, last_seen, notes, tags, custom_fields,
		location, category, primary_role, owner,
		classification_confidence, classification_source, classification_signals,
		parent_device_id, network_layer
		FROM recon_devices WHERE mac_address = ?`, mac))
}

// GetDeviceByIP returns the first device matching the given IP address.
func (s *ReconStore) GetDeviceByIP(ctx context.Context, ip string) (*models.Device, error) {
	// SQLite JSON: check if ip_addresses array contains the IP.
	return s.scanDevice(s.db.QueryRowContext(ctx, `SELECT
		id, hostname, ip_addresses, mac_address, manufacturer,
		device_type, os, status, discovery_method, agent_id,
		first_seen, last_seen, notes, tags, custom_fields,
		location, category, primary_role, owner,
		classification_confidence, classification_source, classification_signals,
		parent_device_id, network_layer
		FROM recon_devices WHERE ip_addresses LIKE ?`, "%\""+ip+"\"%"))
}

// GetDeviceByHostname returns the first device matching the given hostname.
func (s *ReconStore) GetDeviceByHostname(ctx context.Context, hostname string) (*models.Device, error) {
	return s.scanDevice(s.db.QueryRowContext(ctx, `SELECT
		id, hostname, ip_addresses, mac_address, manufacturer,
		device_type, os, status, discovery_method, agent_id,
		first_seen, last_seen, notes, tags, custom_fields,
		location, category, primary_role, owner,
		classification_confidence, classification_source, classification_signals,
		parent_device_id, network_layer
		FROM recon_devices WHERE hostname = ?`, hostname))
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
	if opts.Category != "" {
		where += " AND category = ?"
		args = append(args, opts.Category)
	}
	if opts.Owner != "" {
		where += " AND owner = ?"
		args = append(args, opts.Owner)
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
		"first_seen, last_seen, notes, tags, custom_fields, "+
		"location, category, primary_role, owner, "+
		"classification_confidence, classification_source, classification_signals, "+
		"parent_device_id, network_layer "+
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
		first_seen, last_seen, notes, tags, custom_fields,
		location, category, primary_role, owner,
		classification_confidence, classification_source, classification_signals,
		parent_device_id, network_layer
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
		&d.Location, &d.Category, &d.PrimaryRole, &d.Owner,
		&d.ClassificationConfidence, &d.ClassificationSource, &d.ClassificationSignals,
		&d.ParentDeviceID, &d.NetworkLayer,
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
		&d.Location, &d.Category, &d.PrimaryRole, &d.Owner,
		&d.ClassificationConfidence, &d.ClassificationSource, &d.ClassificationSignals,
		&d.ParentDeviceID, &d.NetworkLayer,
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
			_, err = s.db.ExecContext(ctx, `UPDATE recon_devices SET device_type = ?,
				classification_confidence = 100, classification_source = 'manual', classification_signals = '[]'
				WHERE id = ?`, *params.DeviceType, id)
			if err != nil {
				return fmt.Errorf("update device_type: %w", err)
			}
		}
	}
	if params.Location != nil {
		_, err = s.db.ExecContext(ctx, `UPDATE recon_devices SET location = ? WHERE id = ?`, *params.Location, id)
		if err != nil {
			return fmt.Errorf("update location: %w", err)
		}
	}
	if params.Category != nil {
		_, err = s.db.ExecContext(ctx, `UPDATE recon_devices SET category = ? WHERE id = ?`, *params.Category, id)
		if err != nil {
			return fmt.Errorf("update category: %w", err)
		}
	}
	if params.PrimaryRole != nil {
		_, err = s.db.ExecContext(ctx, `UPDATE recon_devices SET primary_role = ? WHERE id = ?`, *params.PrimaryRole, id)
		if err != nil {
			return fmt.Errorf("update primary_role: %w", err)
		}
	}
	if params.Owner != nil {
		_, err = s.db.ExecContext(ctx, `UPDATE recon_devices SET owner = ? WHERE id = ?`, *params.Owner, id)
		if err != nil {
			return fmt.Errorf("update owner: %w", err)
		}
	}
	return nil
}

// UpdateDeviceType updates just the device_type field for a device.
func (s *ReconStore) UpdateDeviceType(ctx context.Context, deviceID string, deviceType models.DeviceType) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE recon_devices SET device_type = ? WHERE id = ?`,
		string(deviceType), deviceID)
	if err != nil {
		return fmt.Errorf("update device type: %w", err)
	}
	return nil
}

// UpdateDeviceClassification updates the device type along with classification metadata.
func (s *ReconStore) UpdateDeviceClassification(ctx context.Context, deviceID string, deviceType models.DeviceType, confidence int, source, signalsJSON string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE recon_devices SET device_type = ?, classification_confidence = ?, classification_source = ?, classification_signals = ? WHERE id = ?`,
		string(deviceType), confidence, source, signalsJSON, deviceID)
	if err != nil {
		return fmt.Errorf("update device classification: %w", err)
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
			first_seen, last_seen, notes, tags, custom_fields,
			location, category, primary_role, owner,
			classification_confidence, classification_source, classification_signals,
			parent_device_id, network_layer
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		device.ID, device.Hostname, string(ipsJSON), device.MACAddress, device.Manufacturer,
		string(device.DeviceType), device.OS, string(device.Status), string(device.DiscoveryMethod), device.AgentID,
		now, now, device.Notes, string(tagsJSON), string(cfJSON),
		device.Location, device.Category, device.PrimaryRole, device.Owner,
		device.ClassificationConfidence, device.ClassificationSource, device.ClassificationSignals,
		device.ParentDeviceID, device.NetworkLayer,
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

// GetInventorySummary returns aggregate statistics about the device inventory.
// staleDays controls how many days since last_seen a device is considered stale.
func (s *ReconStore) GetInventorySummary(ctx context.Context, staleDays int) (*InventorySummary, error) {
	summary := &InventorySummary{
		ByCategory: make(map[string]int),
		ByType:     make(map[string]int),
	}

	// Total, online, offline counts.
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'online' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'offline' THEN 1 ELSE 0 END), 0)
		FROM recon_devices`,
	).Scan(&summary.TotalDevices, &summary.OnlineCount, &summary.OfflineCount)
	if err != nil {
		return nil, fmt.Errorf("inventory counts: %w", err)
	}

	// Stale count: online devices not seen in staleDays.
	threshold := time.Now().UTC().Add(-time.Duration(staleDays) * 24 * time.Hour)
	err = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM recon_devices WHERE status = 'online' AND last_seen < ?`,
		threshold,
	).Scan(&summary.StaleCount)
	if err != nil {
		return nil, fmt.Errorf("stale count: %w", err)
	}

	// Group by category.
	catRows, err := s.db.QueryContext(ctx,
		`SELECT category, COUNT(*) FROM recon_devices WHERE category != '' GROUP BY category`)
	if err != nil {
		return nil, fmt.Errorf("by category: %w", err)
	}
	defer catRows.Close()
	for catRows.Next() {
		var cat string
		var cnt int
		if err := catRows.Scan(&cat, &cnt); err != nil {
			return nil, fmt.Errorf("scan category row: %w", err)
		}
		summary.ByCategory[cat] = cnt
	}
	if err := catRows.Err(); err != nil {
		return nil, fmt.Errorf("category rows: %w", err)
	}

	// Group by device_type.
	typeRows, err := s.db.QueryContext(ctx,
		`SELECT device_type, COUNT(*) FROM recon_devices GROUP BY device_type`)
	if err != nil {
		return nil, fmt.Errorf("by type: %w", err)
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var dt string
		var cnt int
		if err := typeRows.Scan(&dt, &cnt); err != nil {
			return nil, fmt.Errorf("scan type row: %w", err)
		}
		summary.ByType[dt] = cnt
	}
	if err := typeRows.Err(); err != nil {
		return nil, fmt.Errorf("type rows: %w", err)
	}

	return summary, nil
}

// RemoveARPLinksForDevice removes ARP-inferred topology links for a device
// that now has LLDP-discovered links (LLDP is more accurate).
func (s *ReconStore) RemoveARPLinksForDevice(ctx context.Context, deviceID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM recon_topology_links WHERE link_type = 'arp' AND (source_device_id = ? OR target_device_id = ?)`,
		deviceID, deviceID)
	if err != nil {
		return fmt.Errorf("remove ARP links for device %s: %w", deviceID, err)
	}
	return nil
}

// RemoveFDBLinksForDevice removes FDB-inferred topology links where the given
// device is the source (switch). Called before re-inserting fresh FDB data.
func (s *ReconStore) RemoveFDBLinksForDevice(ctx context.Context, deviceID string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM recon_topology_links WHERE link_type = 'fdb' AND source_device_id = ?`,
		deviceID)
	if err != nil {
		return fmt.Errorf("remove FDB links for device %s: %w", deviceID, err)
	}
	return nil
}

// BulkUpdateDevices applies the same partial update to all devices matching the given IDs.
// Returns the number of updated rows.
func (s *ReconStore) BulkUpdateDevices(ctx context.Context, ids []string, params UpdateDeviceParams) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback on commit is a no-op

	// Build SET clause dynamically.
	setClauses := []string{}
	setArgs := []any{}

	if params.Notes != nil {
		setClauses = append(setClauses, "notes = ?")
		setArgs = append(setArgs, *params.Notes)
	}
	if params.Tags != nil {
		tagsJSON, _ := json.Marshal(*params.Tags)
		setClauses = append(setClauses, "tags = ?")
		setArgs = append(setArgs, string(tagsJSON))
	}
	if params.CustomFields != nil {
		cfJSON, _ := json.Marshal(*params.CustomFields)
		setClauses = append(setClauses, "custom_fields = ?")
		setArgs = append(setArgs, string(cfJSON))
	}
	if params.DeviceType != nil {
		setClauses = append(setClauses, "device_type = ?")
		setArgs = append(setArgs, *params.DeviceType)
	}
	if params.Location != nil {
		setClauses = append(setClauses, "location = ?")
		setArgs = append(setArgs, *params.Location)
	}
	if params.Category != nil {
		setClauses = append(setClauses, "category = ?")
		setArgs = append(setArgs, *params.Category)
	}
	if params.PrimaryRole != nil {
		setClauses = append(setClauses, "primary_role = ?")
		setArgs = append(setArgs, *params.PrimaryRole)
	}
	if params.Owner != nil {
		setClauses = append(setClauses, "owner = ?")
		setArgs = append(setArgs, *params.Owner)
	}

	if len(setClauses) == 0 {
		return 0, nil
	}

	// Build WHERE IN clause with placeholders.
	placeholders := make([]string, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		setArgs = append(setArgs, id)
	}

	query := "UPDATE recon_devices SET " + //nolint:gosec // G202: dynamic SQL uses parameterized placeholders only
		strings.Join(setClauses, ", ") +
		" WHERE id IN (" + strings.Join(placeholders, ", ") + ")"

	res, err := tx.ExecContext(ctx, query, setArgs...)
	if err != nil {
		return 0, fmt.Errorf("bulk update: %w", err)
	}

	n, _ := res.RowsAffected()

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}

	return int(n), nil
}

// CreateTopologyLayout inserts a new topology layout record.
func (s *ReconStore) CreateTopologyLayout(ctx context.Context, layout *TopologyLayout) error {
	if layout.ID == "" {
		layout.ID = uuid.New().String()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	layout.CreatedAt = now
	layout.UpdatedAt = now
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO recon_topology_layouts (id, name, positions, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)`,
		layout.ID, layout.Name, layout.Positions, layout.CreatedAt, layout.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create topology layout: %w", err)
	}
	return nil
}

// ListTopologyLayouts returns all saved topology layouts ordered by most recently updated.
func (s *ReconStore) ListTopologyLayouts(ctx context.Context) ([]TopologyLayout, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, positions, created_at, updated_at
		FROM recon_topology_layouts
		ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list topology layouts: %w", err)
	}
	defer rows.Close()
	var layouts []TopologyLayout
	for rows.Next() {
		var l TopologyLayout
		if err := rows.Scan(&l.ID, &l.Name, &l.Positions, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan topology layout: %w", err)
		}
		layouts = append(layouts, l)
	}
	return layouts, rows.Err()
}

// GetTopologyLayout returns a single topology layout by ID.
func (s *ReconStore) GetTopologyLayout(ctx context.Context, id string) (*TopologyLayout, error) {
	var l TopologyLayout
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, positions, created_at, updated_at
		FROM recon_topology_layouts WHERE id = ?`, id,
	).Scan(&l.ID, &l.Name, &l.Positions, &l.CreatedAt, &l.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get topology layout: %w", err)
	}
	return &l, nil
}

// UpdateTopologyLayout updates an existing topology layout.
func (s *ReconStore) UpdateTopologyLayout(ctx context.Context, layout *TopologyLayout) error {
	now := time.Now().UTC().Format(time.RFC3339)
	layout.UpdatedAt = now
	result, err := s.db.ExecContext(ctx, `
		UPDATE recon_topology_layouts
		SET name = ?, positions = ?, updated_at = ?
		WHERE id = ?`,
		layout.Name, layout.Positions, layout.UpdatedAt, layout.ID,
	)
	if err != nil {
		return fmt.Errorf("update topology layout: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("topology layout not found: %s", layout.ID)
	}
	return nil
}

// SaveScanMetrics inserts timing and count metrics for a completed scan.
func (s *ReconStore) SaveScanMetrics(ctx context.Context, m *models.ScanMetrics) error {
	if m.CreatedAt == "" {
		m.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO recon_scan_metrics (
			scan_id, duration_ms, ping_phase_ms, enrich_phase_ms, post_process_ms,
			hosts_scanned, hosts_alive, devices_created, devices_updated, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ScanID, m.DurationMs, m.PingPhaseMs, m.EnrichPhaseMs, m.PostProcessMs,
		m.HostsScanned, m.HostsAlive, m.DevicesCreated, m.DevicesUpdated, m.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert scan metrics: %w", err)
	}
	return nil
}

// GetScanMetrics returns the metrics for a given scan ID.
// Returns nil without error if no metrics exist for the scan.
func (s *ReconStore) GetScanMetrics(ctx context.Context, scanID string) (*models.ScanMetrics, error) {
	var m models.ScanMetrics
	err := s.db.QueryRowContext(ctx, `
		SELECT scan_id, duration_ms, ping_phase_ms, enrich_phase_ms, post_process_ms,
			hosts_scanned, hosts_alive, devices_created, devices_updated, created_at
		FROM recon_scan_metrics WHERE scan_id = ?`, scanID,
	).Scan(&m.ScanID, &m.DurationMs, &m.PingPhaseMs, &m.EnrichPhaseMs, &m.PostProcessMs,
		&m.HostsScanned, &m.HostsAlive, &m.DevicesCreated, &m.DevicesUpdated, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get scan metrics: %w", err)
	}
	return &m, nil
}

// DeleteTopologyLayout removes a topology layout by ID.
func (s *ReconStore) DeleteTopologyLayout(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM recon_topology_layouts WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("delete topology layout: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("topology layout not found: %s", id)
	}
	return nil
}

// SaveScanMetricsAggregate inserts a pre-computed aggregate record.
// Uses INSERT OR IGNORE to be idempotent against the unique (period, period_start) index.
func (s *ReconStore) SaveScanMetricsAggregate(ctx context.Context, agg *ScanMetricsAggregate) error {
	if agg.ID == "" {
		agg.ID = uuid.New().String()
	}
	if agg.CreatedAt == "" {
		agg.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO recon_scan_metrics_aggregates (
			id, period, period_start, period_end, scan_count,
			avg_duration_ms, avg_ping_phase_ms, avg_enrich_ms,
			avg_devices_found, max_devices_found, min_devices_found,
			avg_hosts_alive, total_new_devices, failed_scans, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agg.ID, agg.Period, agg.PeriodStart, agg.PeriodEnd, agg.ScanCount,
		agg.AvgDurationMs, agg.AvgPingPhaseMs, agg.AvgEnrichMs,
		agg.AvgDevicesFound, agg.MaxDevicesFound, agg.MinDevicesFound,
		agg.AvgHostsAlive, agg.TotalNewDevices, agg.FailedScans, agg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert scan metrics aggregate: %w", err)
	}
	return nil
}

// ListScanMetricsAggregates returns aggregates for the given period, ordered by period_start descending.
func (s *ReconStore) ListScanMetricsAggregates(ctx context.Context, period string, limit int) ([]ScanMetricsAggregate, error) {
	if limit <= 0 {
		limit = 52
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, period, period_start, period_end, scan_count,
			avg_duration_ms, avg_ping_phase_ms, avg_enrich_ms,
			avg_devices_found, max_devices_found, min_devices_found,
			avg_hosts_alive, total_new_devices, failed_scans, created_at
		FROM recon_scan_metrics_aggregates
		WHERE period = ?
		ORDER BY period_start DESC
		LIMIT ?`, period, limit)
	if err != nil {
		return nil, fmt.Errorf("list scan metrics aggregates: %w", err)
	}
	defer rows.Close()

	var result []ScanMetricsAggregate
	for rows.Next() {
		var a ScanMetricsAggregate
		if err := rows.Scan(
			&a.ID, &a.Period, &a.PeriodStart, &a.PeriodEnd, &a.ScanCount,
			&a.AvgDurationMs, &a.AvgPingPhaseMs, &a.AvgEnrichMs,
			&a.AvgDevicesFound, &a.MaxDevicesFound, &a.MinDevicesFound,
			&a.AvgHostsAlive, &a.TotalNewDevices, &a.FailedScans, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan aggregate row: %w", err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

// GetRawMetricsSince returns raw scan metrics with created_at >= since.
func (s *ReconStore) GetRawMetricsSince(ctx context.Context, since time.Time) ([]models.ScanMetrics, error) {
	sinceStr := since.UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
		SELECT scan_id, duration_ms, ping_phase_ms, enrich_phase_ms, post_process_ms,
			hosts_scanned, hosts_alive, devices_created, devices_updated, created_at
		FROM recon_scan_metrics
		WHERE created_at >= ?
		ORDER BY created_at ASC`, sinceStr)
	if err != nil {
		return nil, fmt.Errorf("get raw metrics since: %w", err)
	}
	defer rows.Close()

	var result []models.ScanMetrics
	for rows.Next() {
		var m models.ScanMetrics
		if err := rows.Scan(
			&m.ScanID, &m.DurationMs, &m.PingPhaseMs, &m.EnrichPhaseMs, &m.PostProcessMs,
			&m.HostsScanned, &m.HostsAlive, &m.DevicesCreated, &m.DevicesUpdated, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan raw metrics row: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// GetRawMetricsInRange returns raw scan metrics within the given time range.
func (s *ReconStore) GetRawMetricsInRange(ctx context.Context, start, end time.Time) ([]models.ScanMetrics, error) {
	startStr := start.UTC().Format(time.RFC3339)
	endStr := end.UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
		SELECT scan_id, duration_ms, ping_phase_ms, enrich_phase_ms, post_process_ms,
			hosts_scanned, hosts_alive, devices_created, devices_updated, created_at
		FROM recon_scan_metrics
		WHERE created_at >= ? AND created_at < ?
		ORDER BY created_at ASC`, startStr, endStr)
	if err != nil {
		return nil, fmt.Errorf("get raw metrics in range: %w", err)
	}
	defer rows.Close()

	var result []models.ScanMetrics
	for rows.Next() {
		var m models.ScanMetrics
		if err := rows.Scan(
			&m.ScanID, &m.DurationMs, &m.PingPhaseMs, &m.EnrichPhaseMs, &m.PostProcessMs,
			&m.HostsScanned, &m.HostsAlive, &m.DevicesCreated, &m.DevicesUpdated, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan raw metrics row: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// ListRawMetrics returns the most recent raw scan metrics.
func (s *ReconStore) ListRawMetrics(ctx context.Context, limit int) ([]models.ScanMetrics, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT scan_id, duration_ms, ping_phase_ms, enrich_phase_ms, post_process_ms,
			hosts_scanned, hosts_alive, devices_created, devices_updated, created_at
		FROM recon_scan_metrics
		ORDER BY created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list raw metrics: %w", err)
	}
	defer rows.Close()

	var result []models.ScanMetrics
	for rows.Next() {
		var m models.ScanMetrics
		if err := rows.Scan(
			&m.ScanID, &m.DurationMs, &m.PingPhaseMs, &m.EnrichPhaseMs, &m.PostProcessMs,
			&m.HostsScanned, &m.HostsAlive, &m.DevicesCreated, &m.DevicesUpdated, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan raw metrics row: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// PruneMetricsBefore deletes raw scan metrics older than the given time.
// Returns the number of deleted rows.
func (s *ReconStore) PruneMetricsBefore(ctx context.Context, before time.Time) (int64, error) {
	beforeStr := before.UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM recon_scan_metrics WHERE created_at < ?`, beforeStr)
	if err != nil {
		return 0, fmt.Errorf("prune scan metrics: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}

// GetWeeklyAggregatesInRange returns weekly aggregates within the given time range.
func (s *ReconStore) GetWeeklyAggregatesInRange(ctx context.Context, start, end time.Time) ([]ScanMetricsAggregate, error) {
	startStr := start.UTC().Format(time.RFC3339)
	endStr := end.UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, period, period_start, period_end, scan_count,
			avg_duration_ms, avg_ping_phase_ms, avg_enrich_ms,
			avg_devices_found, max_devices_found, min_devices_found,
			avg_hosts_alive, total_new_devices, failed_scans, created_at
		FROM recon_scan_metrics_aggregates
		WHERE period = 'weekly' AND period_start >= ? AND period_start < ?
		ORDER BY period_start ASC`, startStr, endStr)
	if err != nil {
		return nil, fmt.Errorf("get weekly aggregates in range: %w", err)
	}
	defer rows.Close()

	var result []ScanMetricsAggregate
	for rows.Next() {
		var a ScanMetricsAggregate
		if err := rows.Scan(
			&a.ID, &a.Period, &a.PeriodStart, &a.PeriodEnd, &a.ScanCount,
			&a.AvgDurationMs, &a.AvgPingPhaseMs, &a.AvgEnrichMs,
			&a.AvgDevicesFound, &a.MaxDevicesFound, &a.MinDevicesFound,
			&a.AvgHostsAlive, &a.TotalNewDevices, &a.FailedScans, &a.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan weekly aggregate row: %w", err)
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

// DeviceTreeNode represents a device with hierarchy information for the tree view.
type DeviceTreeNode struct {
	ID             string              `json:"id"`
	Hostname       string              `json:"hostname"`
	DeviceType     models.DeviceType   `json:"device_type"`
	Status         models.DeviceStatus `json:"status"`
	IPAddresses    []string            `json:"ip_addresses"`
	ParentDeviceID string              `json:"parent_device_id,omitempty"`
	NetworkLayer   int                 `json:"network_layer"`
	ChildCount     int                 `json:"child_count"`
}

// UpdateDeviceHierarchy updates the parent and network layer for a single device.
func (s *ReconStore) UpdateDeviceHierarchy(ctx context.Context, deviceID, parentDeviceID string, networkLayer int) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE recon_devices SET parent_device_id = ?, network_layer = ? WHERE id = ?`,
		parentDeviceID, networkLayer, deviceID)
	if err != nil {
		return fmt.Errorf("update device hierarchy: %w", err)
	}
	return nil
}

// GetDeviceTree returns all devices with hierarchy metadata and child counts.
func (s *ReconStore) GetDeviceTree(ctx context.Context) ([]DeviceTreeNode, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.hostname, d.device_type, d.status, d.ip_addresses,
			d.parent_device_id, d.network_layer,
			(SELECT COUNT(*) FROM recon_devices c WHERE c.parent_device_id = d.id) AS child_count
		FROM recon_devices d
		ORDER BY d.network_layer ASC, d.hostname ASC`)
	if err != nil {
		return nil, fmt.Errorf("get device tree: %w", err)
	}
	defer rows.Close()

	var nodes []DeviceTreeNode
	for rows.Next() {
		var n DeviceTreeNode
		var dt, status, ipsJSON string
		if err := rows.Scan(&n.ID, &n.Hostname, &dt, &status, &ipsJSON,
			&n.ParentDeviceID, &n.NetworkLayer, &n.ChildCount); err != nil {
			return nil, fmt.Errorf("scan device tree row: %w", err)
		}
		n.DeviceType = models.DeviceType(dt)
		n.Status = models.DeviceStatus(status)
		_ = json.Unmarshal([]byte(ipsJSON), &n.IPAddresses)
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

// ClearHierarchy resets parent_device_id and network_layer for all devices
// to allow fresh inference. Does not affect manually set parents.
func (s *ReconStore) ClearHierarchy(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE recon_devices SET parent_device_id = '', network_layer = 0`)
	if err != nil {
		return fmt.Errorf("clear hierarchy: %w", err)
	}
	return nil
}

// ListAllDevices returns all devices without pagination (for hierarchy inference).
func (s *ReconStore) ListAllDevices(ctx context.Context) ([]models.Device, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT
		id, hostname, ip_addresses, mac_address, manufacturer,
		device_type, os, status, discovery_method, agent_id,
		first_seen, last_seen, notes, tags, custom_fields,
		location, category, primary_role, owner,
		classification_confidence, classification_source, classification_signals,
		parent_device_id, network_layer
		FROM recon_devices`)
	if err != nil {
		return nil, fmt.Errorf("list all devices: %w", err)
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		d, scanErr := s.scanDeviceRow(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		devices = append(devices, *d)
	}
	return devices, rows.Err()
}
