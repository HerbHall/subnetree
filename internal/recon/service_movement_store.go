package recon

import (
	"context"
	"fmt"
	"time"
)

// SaveServiceMovement persists a detected service movement.
func (s *ReconStore) SaveServiceMovement(ctx context.Context, movement ServiceMovement) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO recon_service_movements (id, port, protocol, service_name, from_device_id, to_device_id, detected_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		movement.ID, movement.Port, movement.Protocol, movement.ServiceName,
		movement.FromDevice, movement.ToDevice,
		movement.DetectedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert service movement: %w", err)
	}
	return nil
}

// ListServiceMovements returns the most recent service movements, ordered by
// detected_at descending.
func (s *ReconStore) ListServiceMovements(ctx context.Context, limit int) ([]ServiceMovement, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, port, protocol, service_name, from_device_id, to_device_id, detected_at
		FROM recon_service_movements
		ORDER BY detected_at DESC
		LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list service movements: %w", err)
	}
	defer rows.Close()

	var movements []ServiceMovement
	for rows.Next() {
		var m ServiceMovement
		var detectedAt string
		if err := rows.Scan(&m.ID, &m.Port, &m.Protocol, &m.ServiceName,
			&m.FromDevice, &m.ToDevice, &detectedAt); err != nil {
			return nil, fmt.Errorf("scan service movement row: %w", err)
		}
		t, err := time.Parse(time.RFC3339, detectedAt)
		if err != nil {
			return nil, fmt.Errorf("parse detected_at: %w", err)
		}
		m.DetectedAt = t
		movements = append(movements, m)
	}
	return movements, rows.Err()
}

// GetPreviousServiceMap builds a map of device_id -> open ports from the
// most recent scan's linked devices. This serves as the "previous" state
// for service movement detection.
//
// Note: The current implementation returns an empty map because the Device
// model does not yet track open ports. When port scanning is added to the
// scan pipeline, this method should query the stored port data for each
// device linked to the last completed scan.
func (s *ReconStore) GetPreviousServiceMap(ctx context.Context) (map[string][]int, error) {
	// Find the most recent completed scan.
	var scanID string
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM recon_scans
		WHERE status = 'completed'
		ORDER BY ended_at DESC
		LIMIT 1`,
	).Scan(&scanID)
	if err != nil {
		// No previous scan -- return empty map.
		return make(map[string][]int), nil //nolint:nilerr // no previous scan is expected
	}

	// Placeholder: return empty map per device until port scanning is implemented.
	// When the scan pipeline stores open ports per device, populate this from
	// a recon_device_ports or similar table.
	_ = scanID
	return make(map[string][]int), nil
}
