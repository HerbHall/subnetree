package svcmap

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
)

// Store provides database operations for the service mapping module.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store and ensures the services table exists.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("svcmap store migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS services (
			id            TEXT PRIMARY KEY,
			name          TEXT NOT NULL,
			display_name  TEXT NOT NULL DEFAULT '',
			service_type  TEXT NOT NULL,
			device_id     TEXT NOT NULL,
			application_id TEXT NOT NULL DEFAULT '',
			status        TEXT NOT NULL DEFAULT 'unknown',
			desired_state TEXT NOT NULL DEFAULT 'monitoring-only',
			ports_json    TEXT NOT NULL DEFAULT '[]',
			cpu_percent   REAL NOT NULL DEFAULT 0,
			memory_bytes  INTEGER NOT NULL DEFAULT 0,
			first_seen    DATETIME NOT NULL,
			last_seen     DATETIME NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_services_device_id ON services(device_id);
		CREATE INDEX IF NOT EXISTS idx_services_service_type ON services(service_type);
	`)
	return err
}

// UpsertService inserts a new service or updates an existing one.
func (s *Store) UpsertService(ctx context.Context, svc *models.Service) error {
	portsJSON, err := json.Marshal(svc.Ports)
	if err != nil {
		return fmt.Errorf("marshal ports: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO services (
			id, name, display_name, service_type, device_id,
			application_id, status, desired_state, ports_json,
			cpu_percent, memory_bytes, first_seen, last_seen
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			name = excluded.name,
			display_name = excluded.display_name,
			service_type = excluded.service_type,
			device_id = excluded.device_id,
			application_id = excluded.application_id,
			status = excluded.status,
			desired_state = excluded.desired_state,
			ports_json = excluded.ports_json,
			cpu_percent = excluded.cpu_percent,
			memory_bytes = excluded.memory_bytes,
			last_seen = excluded.last_seen`,
		svc.ID, svc.Name, svc.DisplayName, svc.ServiceType, svc.DeviceID,
		svc.ApplicationID, svc.Status, svc.DesiredState, string(portsJSON),
		svc.CPUPercent, svc.MemoryBytes, svc.FirstSeen, svc.LastSeen,
	)
	if err != nil {
		return fmt.Errorf("upsert service: %w", err)
	}
	return nil
}

// GetService returns a service by ID. Returns nil, nil if not found.
func (s *Store) GetService(ctx context.Context, id string) (*models.Service, error) {
	var svc models.Service
	var portsJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, display_name, service_type, device_id,
			application_id, status, desired_state, ports_json,
			cpu_percent, memory_bytes, first_seen, last_seen
		FROM services WHERE id = ?`, id,
	).Scan(
		&svc.ID, &svc.Name, &svc.DisplayName, &svc.ServiceType, &svc.DeviceID,
		&svc.ApplicationID, &svc.Status, &svc.DesiredState, &portsJSON,
		&svc.CPUPercent, &svc.MemoryBytes, &svc.FirstSeen, &svc.LastSeen,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get service: %w", err)
	}
	if err := json.Unmarshal([]byte(portsJSON), &svc.Ports); err != nil {
		return nil, fmt.Errorf("unmarshal ports: %w", err)
	}
	return &svc, nil
}

// ListServices returns all tracked services.
func (s *Store) ListServices(ctx context.Context) ([]models.Service, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, display_name, service_type, device_id,
			application_id, status, desired_state, ports_json,
			cpu_percent, memory_bytes, first_seen, last_seen
		FROM services ORDER BY device_id, name`)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()
	return scanServices(rows)
}

// ListServicesByDevice returns all services for a specific device.
func (s *Store) ListServicesByDevice(ctx context.Context, deviceID string) ([]models.Service, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, display_name, service_type, device_id,
			application_id, status, desired_state, ports_json,
			cpu_percent, memory_bytes, first_seen, last_seen
		FROM services WHERE device_id = ? ORDER BY name`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("list services by device: %w", err)
	}
	defer rows.Close()
	return scanServices(rows)
}

// UpdateDesiredState changes the desired state for a service.
func (s *Store) UpdateDesiredState(ctx context.Context, id string, state models.DesiredState) error {
	res, err := s.db.ExecContext(ctx, `UPDATE services SET desired_state = ? WHERE id = ?`, state, id)
	if err != nil {
		return fmt.Errorf("update desired state: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeleteService removes a service by ID.
func (s *Store) DeleteService(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM services WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// MarkStaleServices sets status to "unknown" for services on a device
// that haven't been seen since the given cutoff time.
func (s *Store) MarkStaleServices(ctx context.Context, deviceID string, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE services SET status = ?
		WHERE device_id = ? AND last_seen < ? AND status != ?`,
		models.ServiceStatusUnknown, deviceID, cutoff, models.ServiceStatusUnknown,
	)
	if err != nil {
		return 0, fmt.Errorf("mark stale services: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// FindByDeviceAndName looks up a service by device ID and name.
// Returns nil, nil if not found.
func (s *Store) FindByDeviceAndName(ctx context.Context, deviceID, name string) (*models.Service, error) {
	var svc models.Service
	var portsJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, display_name, service_type, device_id,
			application_id, status, desired_state, ports_json,
			cpu_percent, memory_bytes, first_seen, last_seen
		FROM services WHERE device_id = ? AND name = ?`, deviceID, name,
	).Scan(
		&svc.ID, &svc.Name, &svc.DisplayName, &svc.ServiceType, &svc.DeviceID,
		&svc.ApplicationID, &svc.Status, &svc.DesiredState, &portsJSON,
		&svc.CPUPercent, &svc.MemoryBytes, &svc.FirstSeen, &svc.LastSeen,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find by device and name: %w", err)
	}
	if err := json.Unmarshal([]byte(portsJSON), &svc.Ports); err != nil {
		return nil, fmt.Errorf("unmarshal ports: %w", err)
	}
	return &svc, nil
}

// GetUtilizationSummaries returns per-device resource aggregation.
func (s *Store) GetUtilizationSummaries(ctx context.Context) ([]DeviceServiceStats, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT device_id,
			COUNT(*) as service_count,
			COALESCE(SUM(cpu_percent), 0) as total_cpu,
			COALESCE(SUM(memory_bytes), 0) as total_memory
		FROM services
		WHERE status = ?
		GROUP BY device_id`, models.ServiceStatusRunning)
	if err != nil {
		return nil, fmt.Errorf("get utilization summaries: %w", err)
	}
	defer rows.Close()

	var stats []DeviceServiceStats
	for rows.Next() {
		var ds DeviceServiceStats
		if err := rows.Scan(&ds.DeviceID, &ds.ServiceCount, &ds.TotalCPU, &ds.TotalMemory); err != nil {
			return nil, fmt.Errorf("scan utilization row: %w", err)
		}
		stats = append(stats, ds)
	}
	return stats, rows.Err()
}

// DeviceServiceStats holds aggregated service metrics for a device.
type DeviceServiceStats struct {
	DeviceID     string
	ServiceCount int
	TotalCPU     float64
	TotalMemory  int64
}

// ServiceFilter holds optional filter criteria for listing services.
type ServiceFilter struct {
	DeviceID    string
	ServiceType string
	Status      string
}

// ListServicesFiltered returns services matching the given filter criteria.
func (s *Store) ListServicesFiltered(ctx context.Context, filter ServiceFilter) ([]models.Service, error) {
	query := `SELECT id, name, display_name, service_type, device_id,
		application_id, status, desired_state, ports_json,
		cpu_percent, memory_bytes, first_seen, last_seen
		FROM services WHERE 1=1`
	var args []any

	if filter.DeviceID != "" {
		query += " AND device_id = ?"
		args = append(args, filter.DeviceID)
	}
	if filter.ServiceType != "" {
		query += " AND service_type = ?"
		args = append(args, filter.ServiceType)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	query += " ORDER BY device_id, name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list services filtered: %w", err)
	}
	defer rows.Close()
	return scanServices(rows)
}

// scanServices extracts services from database rows.
func scanServices(rows *sql.Rows) ([]models.Service, error) {
	var services []models.Service
	for rows.Next() {
		var svc models.Service
		var portsJSON string
		if err := rows.Scan(
			&svc.ID, &svc.Name, &svc.DisplayName, &svc.ServiceType, &svc.DeviceID,
			&svc.ApplicationID, &svc.Status, &svc.DesiredState, &portsJSON,
			&svc.CPUPercent, &svc.MemoryBytes, &svc.FirstSeen, &svc.LastSeen,
		); err != nil {
			return nil, fmt.Errorf("scan service row: %w", err)
		}
		if err := json.Unmarshal([]byte(portsJSON), &svc.Ports); err != nil {
			return nil, fmt.Errorf("unmarshal ports: %w", err)
		}
		services = append(services, svc)
	}
	return services, rows.Err()
}
