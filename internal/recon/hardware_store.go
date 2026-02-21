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

// UpsertDeviceHardware inserts or updates a hardware profile for a device.
// If an existing record has collection_source = "manual" and the incoming source
// is not "manual", only empty/zero fields are filled (manual data is preserved).
func (s *ReconStore) UpsertDeviceHardware(ctx context.Context, hw *models.DeviceHardware) error {
	now := time.Now().UTC()

	// Check for existing manual override.
	var existingSource string
	err := s.db.QueryRowContext(ctx,
		"SELECT collection_source FROM recon_device_hardware WHERE device_id = ?",
		hw.DeviceID).Scan(&existingSource)

	if err == nil && existingSource == "manual" && hw.CollectionSource != "manual" {
		// Existing record is manual, incoming is auto-collected.
		// Only update fields that are currently empty/zero.
		_, err = s.db.ExecContext(ctx, `UPDATE recon_device_hardware SET
			hostname = CASE WHEN hostname = '' THEN ? ELSE hostname END,
			fqdn = CASE WHEN fqdn = '' THEN ? ELSE fqdn END,
			os_name = CASE WHEN os_name = '' THEN ? ELSE os_name END,
			os_version = CASE WHEN os_version = '' THEN ? ELSE os_version END,
			os_arch = CASE WHEN os_arch = '' THEN ? ELSE os_arch END,
			kernel = CASE WHEN kernel = '' THEN ? ELSE kernel END,
			cpu_model = CASE WHEN cpu_model = '' THEN ? ELSE cpu_model END,
			cpu_cores = CASE WHEN cpu_cores = 0 THEN ? ELSE cpu_cores END,
			cpu_threads = CASE WHEN cpu_threads = 0 THEN ? ELSE cpu_threads END,
			cpu_arch = CASE WHEN cpu_arch = '' THEN ? ELSE cpu_arch END,
			ram_total_mb = CASE WHEN ram_total_mb = 0 THEN ? ELSE ram_total_mb END,
			ram_type = CASE WHEN ram_type = '' THEN ? ELSE ram_type END,
			ram_slots_used = CASE WHEN ram_slots_used = 0 THEN ? ELSE ram_slots_used END,
			ram_slots_total = CASE WHEN ram_slots_total = 0 THEN ? ELSE ram_slots_total END,
			platform_type = CASE WHEN platform_type = '' THEN ? ELSE platform_type END,
			hypervisor = CASE WHEN hypervisor = '' THEN ? ELSE hypervisor END,
			vm_host_id = CASE WHEN vm_host_id = '' THEN ? ELSE vm_host_id END,
			system_manufacturer = CASE WHEN system_manufacturer = '' THEN ? ELSE system_manufacturer END,
			system_model = CASE WHEN system_model = '' THEN ? ELSE system_model END,
			serial_number = CASE WHEN serial_number = '' THEN ? ELSE serial_number END,
			bios_version = CASE WHEN bios_version = '' THEN ? ELSE bios_version END,
			updated_at = ?
			WHERE device_id = ?`,
			hw.Hostname, hw.FQDN, hw.OSName, hw.OSVersion, hw.OSArch,
			hw.Kernel, hw.CPUModel, hw.CPUCores, hw.CPUThreads, hw.CPUArch,
			hw.RAMTotalMB, hw.RAMType, hw.RAMSlotsUsed, hw.RAMSlotsTotal,
			hw.PlatformType, hw.Hypervisor, hw.VMHostID,
			hw.SystemManufacturer, hw.SystemModel, hw.SerialNumber, hw.BIOSVersion,
			now, hw.DeviceID)
		if err != nil {
			return fmt.Errorf("update hardware (manual override): %w", err)
		}
		return nil
	}

	// Normal upsert (INSERT OR REPLACE).
	collectedAt := &now
	if hw.CollectedAt != nil {
		collectedAt = hw.CollectedAt
	}

	_, err = s.db.ExecContext(ctx, `INSERT OR REPLACE INTO recon_device_hardware (
		device_id, hostname, fqdn, os_name, os_version, os_arch, kernel,
		cpu_model, cpu_cores, cpu_threads, cpu_arch,
		ram_total_mb, ram_type, ram_slots_used, ram_slots_total,
		platform_type, hypervisor, vm_host_id,
		system_manufacturer, system_model, serial_number, bios_version,
		collection_source, collected_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		hw.DeviceID, hw.Hostname, hw.FQDN, hw.OSName, hw.OSVersion, hw.OSArch, hw.Kernel,
		hw.CPUModel, hw.CPUCores, hw.CPUThreads, hw.CPUArch,
		hw.RAMTotalMB, hw.RAMType, hw.RAMSlotsUsed, hw.RAMSlotsTotal,
		hw.PlatformType, hw.Hypervisor, hw.VMHostID,
		hw.SystemManufacturer, hw.SystemModel, hw.SerialNumber, hw.BIOSVersion,
		hw.CollectionSource, collectedAt, now)
	if err != nil {
		return fmt.Errorf("upsert device hardware: %w", err)
	}
	return nil
}

// GetDeviceHardware returns the hardware profile for a device.
func (s *ReconStore) GetDeviceHardware(ctx context.Context, deviceID string) (*models.DeviceHardware, error) {
	var hw models.DeviceHardware
	var collectedAt, updatedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT
		device_id, hostname, fqdn, os_name, os_version, os_arch, kernel,
		cpu_model, cpu_cores, cpu_threads, cpu_arch,
		ram_total_mb, ram_type, ram_slots_used, ram_slots_total,
		platform_type, hypervisor, vm_host_id,
		system_manufacturer, system_model, serial_number, bios_version,
		collection_source, collected_at, updated_at
		FROM recon_device_hardware WHERE device_id = ?`, deviceID).Scan(
		&hw.DeviceID, &hw.Hostname, &hw.FQDN, &hw.OSName, &hw.OSVersion, &hw.OSArch, &hw.Kernel,
		&hw.CPUModel, &hw.CPUCores, &hw.CPUThreads, &hw.CPUArch,
		&hw.RAMTotalMB, &hw.RAMType, &hw.RAMSlotsUsed, &hw.RAMSlotsTotal,
		&hw.PlatformType, &hw.Hypervisor, &hw.VMHostID,
		&hw.SystemManufacturer, &hw.SystemModel, &hw.SerialNumber, &hw.BIOSVersion,
		&hw.CollectionSource, &collectedAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get device hardware: %w", err)
	}
	if collectedAt.Valid {
		hw.CollectedAt = &collectedAt.Time
	}
	if updatedAt.Valid {
		hw.UpdatedAt = &updatedAt.Time
	}
	return &hw, nil
}

// UpsertDeviceStorage replaces auto-collected storage records for a device.
// Manual records (collection_source = "manual") are preserved.
func (s *ReconStore) UpsertDeviceStorage(ctx context.Context, deviceID string, disks []models.DeviceStorage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	// Delete existing non-manual rows for this device.
	_, err = tx.ExecContext(ctx,
		"DELETE FROM recon_device_storage WHERE device_id = ? AND collection_source != 'manual'",
		deviceID)
	if err != nil {
		return fmt.Errorf("delete non-manual storage: %w", err)
	}

	now := time.Now().UTC()
	for i := range disks {
		if disks[i].ID == "" {
			disks[i].ID = uuid.New().String()
		}
		collectedAt := &now
		if disks[i].CollectedAt != nil {
			collectedAt = disks[i].CollectedAt
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO recon_device_storage (
			id, device_id, name, disk_type, interface, capacity_gb, model, role,
			collection_source, collected_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			disks[i].ID, deviceID, disks[i].Name, disks[i].DiskType, disks[i].Interface,
			disks[i].CapacityGB, disks[i].Model, disks[i].Role,
			disks[i].CollectionSource, collectedAt)
		if err != nil {
			return fmt.Errorf("insert storage disk: %w", err)
		}
	}

	return tx.Commit()
}

// GetDeviceStorage returns all storage records for a device.
func (s *ReconStore) GetDeviceStorage(ctx context.Context, deviceID string) ([]models.DeviceStorage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT
		id, device_id, name, disk_type, interface, capacity_gb, model, role,
		collection_source, collected_at
		FROM recon_device_storage WHERE device_id = ?`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device storage: %w", err)
	}
	defer rows.Close()

	var result []models.DeviceStorage
	for rows.Next() {
		var d models.DeviceStorage
		var collectedAt sql.NullTime
		if err := rows.Scan(&d.ID, &d.DeviceID, &d.Name, &d.DiskType, &d.Interface,
			&d.CapacityGB, &d.Model, &d.Role, &d.CollectionSource, &collectedAt); err != nil {
			return nil, fmt.Errorf("scan storage row: %w", err)
		}
		if collectedAt.Valid {
			d.CollectedAt = &collectedAt.Time
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// UpsertDeviceGPU replaces auto-collected GPU records for a device.
// Manual records (collection_source = "manual") are preserved.
func (s *ReconStore) UpsertDeviceGPU(ctx context.Context, deviceID string, gpus []models.DeviceGPU) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	// Delete existing non-manual rows for this device.
	_, err = tx.ExecContext(ctx,
		"DELETE FROM recon_device_gpu WHERE device_id = ? AND collection_source != 'manual'",
		deviceID)
	if err != nil {
		return fmt.Errorf("delete non-manual gpu: %w", err)
	}

	now := time.Now().UTC()
	for i := range gpus {
		if gpus[i].ID == "" {
			gpus[i].ID = uuid.New().String()
		}
		collectedAt := &now
		if gpus[i].CollectedAt != nil {
			collectedAt = gpus[i].CollectedAt
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO recon_device_gpu (
			id, device_id, model, vendor, vram_mb, driver_version,
			collection_source, collected_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			gpus[i].ID, deviceID, gpus[i].Model, gpus[i].Vendor,
			gpus[i].VRAMMB, gpus[i].DriverVersion,
			gpus[i].CollectionSource, collectedAt)
		if err != nil {
			return fmt.Errorf("insert gpu: %w", err)
		}
	}

	return tx.Commit()
}

// GetDeviceGPU returns all GPU records for a device.
func (s *ReconStore) GetDeviceGPU(ctx context.Context, deviceID string) ([]models.DeviceGPU, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT
		id, device_id, model, vendor, vram_mb, driver_version,
		collection_source, collected_at
		FROM recon_device_gpu WHERE device_id = ?`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device gpu: %w", err)
	}
	defer rows.Close()

	var result []models.DeviceGPU
	for rows.Next() {
		var g models.DeviceGPU
		var collectedAt sql.NullTime
		if err := rows.Scan(&g.ID, &g.DeviceID, &g.Model, &g.Vendor,
			&g.VRAMMB, &g.DriverVersion, &g.CollectionSource, &collectedAt); err != nil {
			return nil, fmt.Errorf("scan gpu row: %w", err)
		}
		if collectedAt.Valid {
			g.CollectedAt = &collectedAt.Time
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

// UpsertDeviceServices replaces auto-collected service records for a device.
// Manual records (collection_source = "manual") are preserved.
func (s *ReconStore) UpsertDeviceServices(ctx context.Context, deviceID string, svcs []models.DeviceService) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	// Delete existing non-manual rows for this device.
	_, err = tx.ExecContext(ctx,
		"DELETE FROM recon_device_services WHERE device_id = ? AND collection_source != 'manual'",
		deviceID)
	if err != nil {
		return fmt.Errorf("delete non-manual services: %w", err)
	}

	now := time.Now().UTC()
	for i := range svcs {
		if svcs[i].ID == "" {
			svcs[i].ID = uuid.New().String()
		}
		collectedAt := &now
		if svcs[i].CollectedAt != nil {
			collectedAt = svcs[i].CollectedAt
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO recon_device_services (
			id, device_id, name, service_type, port, url, version, status,
			collection_source, collected_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			svcs[i].ID, deviceID, svcs[i].Name, svcs[i].ServiceType,
			svcs[i].Port, svcs[i].URL, svcs[i].Version, svcs[i].Status,
			svcs[i].CollectionSource, collectedAt)
		if err != nil {
			return fmt.Errorf("insert service: %w", err)
		}
	}

	return tx.Commit()
}

// GetDeviceServices returns all service records for a device.
func (s *ReconStore) GetDeviceServices(ctx context.Context, deviceID string) ([]models.DeviceService, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT
		id, device_id, name, service_type, port, url, version, status,
		collection_source, collected_at
		FROM recon_device_services WHERE device_id = ?`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("get device services: %w", err)
	}
	defer rows.Close()

	var result []models.DeviceService
	for rows.Next() {
		var svc models.DeviceService
		var collectedAt sql.NullTime
		if err := rows.Scan(&svc.ID, &svc.DeviceID, &svc.Name, &svc.ServiceType,
			&svc.Port, &svc.URL, &svc.Version, &svc.Status,
			&svc.CollectionSource, &collectedAt); err != nil {
			return nil, fmt.Errorf("scan service row: %w", err)
		}
		if collectedAt.Valid {
			svc.CollectedAt = &collectedAt.Time
		}
		result = append(result, svc)
	}
	return result, rows.Err()
}

// GetHardwareSummary returns fleet-wide aggregate hardware statistics.
func (s *ReconStore) GetHardwareSummary(ctx context.Context) (*models.HardwareSummary, error) {
	summary := &models.HardwareSummary{
		ByOS:           make(map[string]int),
		ByCPUModel:     make(map[string]int),
		ByPlatformType: make(map[string]int),
		ByGPUVendor:    make(map[string]int),
	}

	// Total devices with hardware profiles and total RAM.
	err := s.db.QueryRowContext(ctx, `SELECT
		COUNT(*), COALESCE(SUM(ram_total_mb), 0)
		FROM recon_device_hardware`).Scan(&summary.TotalWithHardware, &summary.TotalRAMMB)
	if err != nil {
		return nil, fmt.Errorf("hardware totals: %w", err)
	}

	// Total storage.
	err = s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(capacity_gb), 0) FROM recon_device_storage`).Scan(&summary.TotalStorageGB)
	if err != nil {
		return nil, fmt.Errorf("storage total: %w", err)
	}

	// Total GPUs.
	err = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM recon_device_gpu`).Scan(&summary.TotalGPUs)
	if err != nil {
		return nil, fmt.Errorf("gpu count: %w", err)
	}

	// Group by OS.
	osRows, err := s.db.QueryContext(ctx,
		`SELECT os_name, COUNT(*) FROM recon_device_hardware WHERE os_name != '' GROUP BY os_name`)
	if err != nil {
		return nil, fmt.Errorf("by os: %w", err)
	}
	defer osRows.Close()
	for osRows.Next() {
		var name string
		var cnt int
		if err := osRows.Scan(&name, &cnt); err != nil {
			return nil, fmt.Errorf("scan os row: %w", err)
		}
		summary.ByOS[name] = cnt
	}
	if err := osRows.Err(); err != nil {
		return nil, fmt.Errorf("os rows: %w", err)
	}

	// Group by CPU model.
	cpuRows, err := s.db.QueryContext(ctx,
		`SELECT cpu_model, COUNT(*) FROM recon_device_hardware WHERE cpu_model != '' GROUP BY cpu_model`)
	if err != nil {
		return nil, fmt.Errorf("by cpu: %w", err)
	}
	defer cpuRows.Close()
	for cpuRows.Next() {
		var model string
		var cnt int
		if err := cpuRows.Scan(&model, &cnt); err != nil {
			return nil, fmt.Errorf("scan cpu row: %w", err)
		}
		summary.ByCPUModel[model] = cnt
	}
	if err := cpuRows.Err(); err != nil {
		return nil, fmt.Errorf("cpu rows: %w", err)
	}

	// Group by platform type.
	platRows, err := s.db.QueryContext(ctx,
		`SELECT platform_type, COUNT(*) FROM recon_device_hardware WHERE platform_type != '' GROUP BY platform_type`)
	if err != nil {
		return nil, fmt.Errorf("by platform: %w", err)
	}
	defer platRows.Close()
	for platRows.Next() {
		var pt string
		var cnt int
		if err := platRows.Scan(&pt, &cnt); err != nil {
			return nil, fmt.Errorf("scan platform row: %w", err)
		}
		summary.ByPlatformType[pt] = cnt
	}
	if err := platRows.Err(); err != nil {
		return nil, fmt.Errorf("platform rows: %w", err)
	}

	// Group by GPU vendor.
	gpuRows, err := s.db.QueryContext(ctx,
		`SELECT vendor, COUNT(*) FROM recon_device_gpu WHERE vendor != '' GROUP BY vendor`)
	if err != nil {
		return nil, fmt.Errorf("by gpu vendor: %w", err)
	}
	defer gpuRows.Close()
	for gpuRows.Next() {
		var vendor string
		var cnt int
		if err := gpuRows.Scan(&vendor, &cnt); err != nil {
			return nil, fmt.Errorf("scan gpu vendor row: %w", err)
		}
		summary.ByGPUVendor[vendor] = cnt
	}
	if err := gpuRows.Err(); err != nil {
		return nil, fmt.Errorf("gpu vendor rows: %w", err)
	}

	return summary, nil
}

// QueryDevicesByHardware returns devices matching hardware filters with pagination.
// Returns matching devices, total count, and any error.
func (s *ReconStore) QueryDevicesByHardware(ctx context.Context, q models.HardwareQuery) ([]models.Device, int, error) {
	if q.Limit <= 0 {
		q.Limit = 50
	}

	where := []string{"1=1"}
	args := []any{}

	if q.MinRAMMB > 0 {
		where = append(where, "h.ram_total_mb >= ?")
		args = append(args, q.MinRAMMB)
	}
	if q.MaxRAMMB > 0 {
		where = append(where, "h.ram_total_mb <= ?")
		args = append(args, q.MaxRAMMB)
	}
	if q.CPUModel != "" {
		where = append(where, "h.cpu_model LIKE ?")
		args = append(args, "%"+q.CPUModel+"%")
	}
	if q.OSName != "" {
		where = append(where, "h.os_name LIKE ?")
		args = append(args, "%"+q.OSName+"%")
	}
	if q.PlatformType != "" {
		where = append(where, "h.platform_type = ?")
		args = append(args, q.PlatformType)
	}
	if q.HasGPU != nil && *q.HasGPU {
		where = append(where, "EXISTS (SELECT 1 FROM recon_device_gpu g WHERE g.device_id = d.id)")
	}
	if q.HasGPU != nil && !*q.HasGPU {
		where = append(where, "NOT EXISTS (SELECT 1 FROM recon_device_gpu g WHERE g.device_id = d.id)")
	}
	if q.GPUVendor != "" {
		where = append(where, "EXISTS (SELECT 1 FROM recon_device_gpu g WHERE g.device_id = d.id AND g.vendor = ?)")
		args = append(args, q.GPUVendor)
	}

	whereClause := strings.Join(where, " AND ")

	// Count total.
	var total int
	countQuery := "SELECT COUNT(*) FROM recon_devices d " +
		"JOIN recon_device_hardware h ON h.device_id = d.id " +
		"WHERE " + whereClause //nolint:gosec // where uses parameterized placeholders only
	err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count hardware query: %w", err)
	}

	// Query with pagination.
	queryArgs := make([]any, 0, len(args)+2)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, q.Limit, q.Offset)

	dataQuery := "SELECT " + //nolint:gosec // where uses parameterized placeholders only
		"d.id, d.hostname, d.ip_addresses, d.mac_address, d.manufacturer, " +
		"d.device_type, d.os, d.status, d.discovery_method, d.agent_id, " +
		"d.first_seen, d.last_seen, d.notes, d.tags, d.custom_fields, " +
		"d.location, d.category, d.primary_role, d.owner, " +
		"d.classification_confidence, d.classification_source, d.classification_signals, " +
		"d.parent_device_id, d.network_layer " +
		"FROM recon_devices d " +
		"JOIN recon_device_hardware h ON h.device_id = d.id " +
		"WHERE " + whereClause + " ORDER BY d.last_seen DESC LIMIT ? OFFSET ?"

	rows, err := s.db.QueryContext(ctx, dataQuery, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query devices by hardware: %w", err)
	}
	defer rows.Close()

	var devices []models.Device
	for rows.Next() {
		var d models.Device
		var ipsJSON, tagsJSON, cfJSON string
		var dt, status, method string
		err := rows.Scan(
			&d.ID, &d.Hostname, &ipsJSON, &d.MACAddress, &d.Manufacturer,
			&dt, &d.OS, &status, &method, &d.AgentID,
			&d.FirstSeen, &d.LastSeen, &d.Notes, &tagsJSON, &cfJSON,
			&d.Location, &d.Category, &d.PrimaryRole, &d.Owner,
			&d.ClassificationConfidence, &d.ClassificationSource, &d.ClassificationSignals,
			&d.ParentDeviceID, &d.NetworkLayer)
		if err != nil {
			return nil, 0, fmt.Errorf("scan device row: %w", err)
		}
		d.DeviceType = models.DeviceType(dt)
		d.Status = models.DeviceStatus(status)
		d.DiscoveryMethod = models.DiscoveryMethod(method)
		_ = json.Unmarshal([]byte(ipsJSON), &d.IPAddresses)
		_ = json.Unmarshal([]byte(tagsJSON), &d.Tags)
		_ = json.Unmarshal([]byte(cfJSON), &d.CustomFields)
		devices = append(devices, d)
	}
	return devices, total, rows.Err()
}

// DeleteDeviceHardware removes the hardware profile for a device.
func (s *ReconStore) DeleteDeviceHardware(ctx context.Context, deviceID string) error {
	res, err := s.db.ExecContext(ctx,
		"DELETE FROM recon_device_hardware WHERE device_id = ?", deviceID)
	if err != nil {
		return fmt.Errorf("delete device hardware: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
