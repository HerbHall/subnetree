package pulse

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Check represents a registered monitoring target.
type Check struct {
	ID              string    `json:"id"`
	DeviceID        string    `json:"device_id"`
	CheckType       string    `json:"check_type"`
	Target          string    `json:"target"`
	IntervalSeconds int       `json:"interval_seconds"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CheckResult represents the outcome of a single health check.
type CheckResult struct {
	ID           int64     `json:"id"`
	CheckID      string    `json:"check_id"`
	DeviceID     string    `json:"device_id"`
	Success      bool      `json:"success"`
	LatencyMs    float64   `json:"latency_ms"`
	PacketLoss   float64   `json:"packet_loss"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CheckedAt    time.Time `json:"checked_at"`
}

// Alert represents a triggered monitoring alert.
type Alert struct {
	ID                  string     `json:"id"`
	CheckID             string     `json:"check_id"`
	DeviceID            string     `json:"device_id"`
	Severity            string     `json:"severity"`
	Message             string     `json:"message"`
	TriggeredAt         time.Time  `json:"triggered_at"`
	ResolvedAt          *time.Time `json:"resolved_at,omitempty"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
}

// PulseStore provides database access for the Pulse monitoring plugin.
type PulseStore struct {
	db *sql.DB
}

// NewPulseStore creates a new PulseStore backed by the given database.
func NewPulseStore(db *sql.DB) *PulseStore {
	return &PulseStore{db: db}
}

// -- Checks --

// InsertCheck inserts a new monitoring check.
func (s *PulseStore) InsertCheck(ctx context.Context, c *Check) error {
	enabled := 0
	if c.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pulse_checks (
			id, device_id, check_type, target, interval_seconds, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.DeviceID, c.CheckType, c.Target, c.IntervalSeconds,
		enabled, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert check: %w", err)
	}
	return nil
}

// GetCheck returns a check by ID. Returns nil, nil if not found.
func (s *PulseStore) GetCheck(ctx context.Context, id string) (*Check, error) {
	var c Check
	var enabledInt int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, device_id, check_type, target, interval_seconds, enabled, created_at, updated_at
		FROM pulse_checks WHERE id = ?`,
		id,
	).Scan(
		&c.ID, &c.DeviceID, &c.CheckType, &c.Target, &c.IntervalSeconds,
		&enabledInt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get check: %w", err)
	}
	c.Enabled = enabledInt != 0
	return &c, nil
}

// GetCheckByDeviceID returns the first check for a device. Returns nil, nil if not found.
func (s *PulseStore) GetCheckByDeviceID(ctx context.Context, deviceID string) (*Check, error) {
	var c Check
	var enabledInt int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, device_id, check_type, target, interval_seconds, enabled, created_at, updated_at
		FROM pulse_checks WHERE device_id = ? LIMIT 1`,
		deviceID,
	).Scan(
		&c.ID, &c.DeviceID, &c.CheckType, &c.Target, &c.IntervalSeconds,
		&enabledInt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get check by device_id: %w", err)
	}
	c.Enabled = enabledInt != 0
	return &c, nil
}

// ListEnabledChecks returns all enabled monitoring checks.
func (s *PulseStore) ListEnabledChecks(ctx context.Context) ([]Check, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, device_id, check_type, target, interval_seconds, enabled, created_at, updated_at
		FROM pulse_checks WHERE enabled = 1 ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled checks: %w", err)
	}
	defer rows.Close()

	var checks []Check
	for rows.Next() {
		var c Check
		var enabledInt int
		if err := rows.Scan(
			&c.ID, &c.DeviceID, &c.CheckType, &c.Target, &c.IntervalSeconds,
			&enabledInt, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan check row: %w", err)
		}
		c.Enabled = enabledInt != 0
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

// UpdateCheckEnabled sets the enabled state of a check.
func (s *PulseStore) UpdateCheckEnabled(ctx context.Context, id string, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE pulse_checks SET enabled = ?, updated_at = ? WHERE id = ?`,
		enabledInt, time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("update check enabled: %w", err)
	}
	return nil
}

// -- Results --

// InsertResult inserts a check result.
func (s *PulseStore) InsertResult(ctx context.Context, r *CheckResult) error {
	success := 0
	if r.Success {
		success = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pulse_check_results (
			check_id, device_id, success, latency_ms, packet_loss, error_message, checked_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		r.CheckID, r.DeviceID, success, r.LatencyMs, r.PacketLoss,
		r.ErrorMessage, r.CheckedAt,
	)
	if err != nil {
		return fmt.Errorf("insert result: %w", err)
	}
	return nil
}

// ListResults returns check results for a device, ordered by checked_at descending.
// If limit <= 0, defaults to 100.
func (s *PulseStore) ListResults(ctx context.Context, deviceID string, limit int) ([]CheckResult, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, check_id, device_id, success, latency_ms, packet_loss, error_message, checked_at
		FROM pulse_check_results WHERE device_id = ? ORDER BY checked_at DESC LIMIT ?`,
		deviceID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list results: %w", err)
	}
	defer rows.Close()

	var results []CheckResult
	for rows.Next() {
		var r CheckResult
		var successInt int
		if err := rows.Scan(
			&r.ID, &r.CheckID, &r.DeviceID, &successInt, &r.LatencyMs,
			&r.PacketLoss, &r.ErrorMessage, &r.CheckedAt,
		); err != nil {
			return nil, fmt.Errorf("scan result row: %w", err)
		}
		r.Success = successInt != 0
		results = append(results, r)
	}
	return results, rows.Err()
}

// DeleteOldResults deletes check results older than the given time.
// Returns the number of rows deleted.
func (s *PulseStore) DeleteOldResults(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM pulse_check_results WHERE checked_at < ?`,
		before,
	)
	if err != nil {
		return 0, fmt.Errorf("delete old results: %w", err)
	}
	return result.RowsAffected()
}

// -- Alerts --

// InsertAlert inserts a new monitoring alert.
func (s *PulseStore) InsertAlert(ctx context.Context, a *Alert) error {
	var resolvedAt sql.NullTime
	if a.ResolvedAt != nil {
		resolvedAt = sql.NullTime{Time: *a.ResolvedAt, Valid: true}
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pulse_alerts (
			id, check_id, device_id, severity, message, triggered_at, resolved_at, consecutive_failures
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.CheckID, a.DeviceID, a.Severity, a.Message,
		a.TriggeredAt, resolvedAt, a.ConsecutiveFailures,
	)
	if err != nil {
		return fmt.Errorf("insert alert: %w", err)
	}
	return nil
}

// ResolveAlert marks an alert as resolved.
func (s *PulseStore) ResolveAlert(ctx context.Context, id string, resolvedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE pulse_alerts SET resolved_at = ? WHERE id = ?`,
		resolvedAt, id,
	)
	if err != nil {
		return fmt.Errorf("resolve alert: %w", err)
	}
	return nil
}

// GetActiveAlert returns the active (unresolved) alert for a check. Returns nil, nil if none.
func (s *PulseStore) GetActiveAlert(ctx context.Context, checkID string) (*Alert, error) {
	var a Alert
	var resolvedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, check_id, device_id, severity, message, triggered_at, resolved_at, consecutive_failures
		FROM pulse_alerts WHERE check_id = ? AND resolved_at IS NULL`,
		checkID,
	).Scan(
		&a.ID, &a.CheckID, &a.DeviceID, &a.Severity, &a.Message,
		&a.TriggeredAt, &resolvedAt, &a.ConsecutiveFailures,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get active alert: %w", err)
	}
	if resolvedAt.Valid {
		a.ResolvedAt = &resolvedAt.Time
	}
	return &a, nil
}

// ListActiveAlerts returns active (unresolved) alerts. If deviceID is empty, returns all
// active alerts. If deviceID is provided, returns only active alerts for that device.
func (s *PulseStore) ListActiveAlerts(ctx context.Context, deviceID string) ([]Alert, error) {
	var rows *sql.Rows
	var err error
	if deviceID == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, check_id, device_id, severity, message, triggered_at, resolved_at, consecutive_failures
			FROM pulse_alerts WHERE resolved_at IS NULL ORDER BY triggered_at DESC`,
		)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, check_id, device_id, severity, message, triggered_at, resolved_at, consecutive_failures
			FROM pulse_alerts WHERE device_id = ? AND resolved_at IS NULL ORDER BY triggered_at DESC`,
			deviceID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("list active alerts: %w", err)
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		var resolvedAt sql.NullTime
		if err := rows.Scan(
			&a.ID, &a.CheckID, &a.DeviceID, &a.Severity, &a.Message,
			&a.TriggeredAt, &resolvedAt, &a.ConsecutiveFailures,
		); err != nil {
			return nil, fmt.Errorf("scan alert row: %w", err)
		}
		if resolvedAt.Valid {
			a.ResolvedAt = &resolvedAt.Time
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

// DeleteOldAlerts deletes resolved alerts older than the given time.
// Returns the number of rows deleted.
func (s *PulseStore) DeleteOldAlerts(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM pulse_alerts WHERE resolved_at IS NOT NULL AND resolved_at < ?`,
		before,
	)
	if err != nil {
		return 0, fmt.Errorf("delete old alerts: %w", err)
	}
	return result.RowsAffected()
}
