package recon

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
)

// ProxmoxResource represents a VM/container resource snapshot joined with device info.
type ProxmoxResource struct {
	DeviceID    string    `json:"device_id"`
	DeviceName  string    `json:"device_name"`
	DeviceType  string    `json:"device_type"`
	Status      string    `json:"status"`
	CPUPercent  float64   `json:"cpu_percent"`
	MemUsedMB   int       `json:"mem_used_mb"`
	MemTotalMB  int       `json:"mem_total_mb"`
	DiskUsedGB  int       `json:"disk_used_gb"`
	DiskTotalGB int       `json:"disk_total_gb"`
	UptimeSec   int64     `json:"uptime_sec"`
	NetInBytes  int64     `json:"netin_bytes"`
	NetOutBytes int64     `json:"netout_bytes"`
	CollectedAt time.Time `json:"collected_at"`
}

// UpsertProxmoxResource inserts or replaces a resource snapshot for a VM/container.
func (s *ReconStore) UpsertProxmoxResource(ctx context.Context, r *ProxmoxResource) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO recon_proxmox_resources (
		device_id, cpu_percent, mem_used_mb, mem_total_mb,
		disk_used_gb, disk_total_gb, uptime_sec,
		netin_bytes, netout_bytes, collected_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.DeviceID, r.CPUPercent, r.MemUsedMB, r.MemTotalMB,
		r.DiskUsedGB, r.DiskTotalGB, r.UptimeSec,
		r.NetInBytes, r.NetOutBytes, r.CollectedAt)
	if err != nil {
		return fmt.Errorf("upsert proxmox resource: %w", err)
	}
	return nil
}

// GetProxmoxResource returns the resource snapshot for a specific VM/container device.
func (s *ReconStore) GetProxmoxResource(ctx context.Context, deviceID string) (*ProxmoxResource, error) {
	var r ProxmoxResource
	err := s.db.QueryRowContext(ctx, `SELECT
		pr.device_id, d.hostname, d.device_type, d.status,
		pr.cpu_percent, pr.mem_used_mb, pr.mem_total_mb,
		pr.disk_used_gb, pr.disk_total_gb, pr.uptime_sec,
		pr.netin_bytes, pr.netout_bytes, pr.collected_at
		FROM recon_proxmox_resources pr
		JOIN recon_devices d ON d.id = pr.device_id
		WHERE pr.device_id = ?`, deviceID).Scan(
		&r.DeviceID, &r.DeviceName, &r.DeviceType, &r.Status,
		&r.CPUPercent, &r.MemUsedMB, &r.MemTotalMB,
		&r.DiskUsedGB, &r.DiskTotalGB, &r.UptimeSec,
		&r.NetInBytes, &r.NetOutBytes, &r.CollectedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get proxmox resource: %w", err)
	}
	return &r, nil
}

// ListProxmoxResources returns resource snapshots for VMs/containers under a parent host.
// If parentDeviceID is empty, returns all Proxmox resources.
// An optional status filter can restrict results to "online" or "offline" devices.
func (s *ReconStore) ListProxmoxResources(ctx context.Context, parentDeviceID, statusFilter string) ([]ProxmoxResource, error) {
	query := `SELECT
		pr.device_id, d.hostname, d.device_type, d.status,
		pr.cpu_percent, pr.mem_used_mb, pr.mem_total_mb,
		pr.disk_used_gb, pr.disk_total_gb, pr.uptime_sec,
		pr.netin_bytes, pr.netout_bytes, pr.collected_at
		FROM recon_proxmox_resources pr
		JOIN recon_devices d ON d.id = pr.device_id
		WHERE 1=1`

	var args []any
	if parentDeviceID != "" {
		query += " AND d.parent_device_id = ?"
		args = append(args, parentDeviceID)
	}
	if statusFilter != "" {
		query += " AND d.status = ?"
		args = append(args, statusFilter)
	}
	query += " ORDER BY d.hostname"

	rows, err := s.db.QueryContext(ctx, query, args...) //nolint:gosec // query uses parameterized placeholders only
	if err != nil {
		return nil, fmt.Errorf("list proxmox resources: %w", err)
	}
	defer rows.Close()

	var result []ProxmoxResource
	for rows.Next() {
		var r ProxmoxResource
		if err := rows.Scan(
			&r.DeviceID, &r.DeviceName, &r.DeviceType, &r.Status,
			&r.CPUPercent, &r.MemUsedMB, &r.MemTotalMB,
			&r.DiskUsedGB, &r.DiskTotalGB, &r.UptimeSec,
			&r.NetInBytes, &r.NetOutBytes, &r.CollectedAt,
		); err != nil {
			return nil, fmt.Errorf("scan proxmox resource row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// FindDeviceByHostnameAndParent looks up a device by hostname and parent_device_id.
// Returns nil, nil when no match is found.
func (s *ReconStore) FindDeviceByHostnameAndParent(ctx context.Context, hostname, parentID string) (*models.Device, error) {
	var d models.Device
	var ipsJSON, tagsJSON, cfJSON string
	var dt, status, method string
	err := s.db.QueryRowContext(ctx, `SELECT
		id, hostname, ip_addresses, mac_address, manufacturer,
		device_type, os, status, discovery_method, agent_id,
		first_seen, last_seen, notes, tags, custom_fields,
		location, category, primary_role, owner,
		classification_confidence, classification_source, classification_signals,
		parent_device_id, network_layer, connection_type
		FROM recon_devices WHERE hostname = ? AND parent_device_id = ?`,
		hostname, parentID).Scan(
		&d.ID, &d.Hostname, &ipsJSON, &d.MACAddress, &d.Manufacturer,
		&dt, &d.OS, &status, &method, &d.AgentID,
		&d.FirstSeen, &d.LastSeen, &d.Notes, &tagsJSON, &cfJSON,
		&d.Location, &d.Category, &d.PrimaryRole, &d.Owner,
		&d.ClassificationConfidence, &d.ClassificationSource, &d.ClassificationSignals,
		&d.ParentDeviceID, &d.NetworkLayer, &d.ConnectionType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find device by hostname+parent: %w", err)
	}
	d.DeviceType = models.DeviceType(dt)
	d.Status = models.DeviceStatus(status)
	d.DiscoveryMethod = models.DiscoveryMethod(method)
	_ = json.Unmarshal([]byte(ipsJSON), &d.IPAddresses)
	_ = json.Unmarshal([]byte(tagsJSON), &d.Tags)
	_ = json.Unmarshal([]byte(cfJSON), &d.CustomFields)
	return &d, nil
}

// FindChildDevicesByDiscovery returns all child devices under a parent that
// were discovered by the given method.
func (s *ReconStore) FindChildDevicesByDiscovery(ctx context.Context, parentID, discoveryMethod string) ([]models.Device, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT
		id, hostname, ip_addresses, mac_address, manufacturer,
		device_type, os, status, discovery_method, agent_id,
		first_seen, last_seen, notes, tags, custom_fields,
		location, category, primary_role, owner,
		classification_confidence, classification_source, classification_signals,
		parent_device_id, network_layer, connection_type
		FROM recon_devices WHERE parent_device_id = ? AND discovery_method = ?`,
		parentID, discoveryMethod)
	if err != nil {
		return nil, fmt.Errorf("find child devices by discovery: %w", err)
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var d models.Device
		var ipsJSON, tagsJSON, cfJSON string
		var dt, status, method string
		if err := rows.Scan(
			&d.ID, &d.Hostname, &ipsJSON, &d.MACAddress, &d.Manufacturer,
			&dt, &d.OS, &status, &method, &d.AgentID,
			&d.FirstSeen, &d.LastSeen, &d.Notes, &tagsJSON, &cfJSON,
			&d.Location, &d.Category, &d.PrimaryRole, &d.Owner,
			&d.ClassificationConfidence, &d.ClassificationSource, &d.ClassificationSignals,
			&d.ParentDeviceID, &d.NetworkLayer, &d.ConnectionType,
		); err != nil {
			return nil, fmt.Errorf("scan child device row: %w", err)
		}
		d.DeviceType = models.DeviceType(dt)
		d.Status = models.DeviceStatus(status)
		d.DiscoveryMethod = models.DiscoveryMethod(method)
		_ = json.Unmarshal([]byte(ipsJSON), &d.IPAddresses)
		_ = json.Unmarshal([]byte(tagsJSON), &d.Tags)
		_ = json.Unmarshal([]byte(cfJSON), &d.CustomFields)
		devices = append(devices, d)
	}
	return devices, rows.Err()
}
