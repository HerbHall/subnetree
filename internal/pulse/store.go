package pulse

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// MetricDataPoint represents a single aggregated metric value at a point in time.
type MetricDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// MetricSeries represents a named time-series of aggregated metric data.
type MetricSeries struct {
	DeviceID string            `json:"device_id"`
	Metric   string            `json:"metric"`
	Range    string            `json:"range"`
	Points   []MetricDataPoint `json:"points"`
}

// Check represents a registered monitoring target.
type Check struct {
	ID              string    `json:"id"`
	DeviceID        string    `json:"device_id"`
	DeviceName      string    `json:"device_name"`
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
	DeviceName          string     `json:"device_name"`
	Severity            string     `json:"severity"`
	Message             string     `json:"message"`
	TriggeredAt         time.Time  `json:"triggered_at"`
	ResolvedAt          *time.Time `json:"resolved_at,omitempty"`
	AcknowledgedAt      *time.Time `json:"acknowledged_at,omitempty"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	Suppressed          bool       `json:"suppressed"`
	SuppressedBy        string     `json:"suppressed_by,omitempty"`
}

// CheckDependency represents a dependency between a check and an upstream device.
// When the upstream device has an active critical alert, the check is suppressed.
type CheckDependency struct {
	CheckID           string    `json:"check_id"`
	DependsOnDeviceID string    `json:"depends_on_device_id"`
	CreatedAt         time.Time `json:"created_at"`
}

// AlertFilters controls filtering for ListAlerts queries.
type AlertFilters struct {
	DeviceID      string
	Severity      string
	ActiveOnly    bool
	Suppressed    *bool // nil = no filter, true = only suppressed, false = only non-suppressed
	Limit         int
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

// ListAllChecks returns all monitoring checks (enabled and disabled).
// Device names are resolved via LEFT JOIN with recon_devices, falling back to
// the first IP address or the raw device_id when hostname is empty.
func (s *PulseStore) ListAllChecks(ctx context.Context) ([]Check, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.device_id, c.check_type, c.target, c.interval_seconds,
			c.enabled, c.created_at, c.updated_at,
			COALESCE(NULLIF(d.hostname, ''), json_extract(d.ip_addresses, '$[0]'), c.device_id) AS device_name
		FROM pulse_checks c
		LEFT JOIN recon_devices d ON d.id = c.device_id
		ORDER BY c.created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("list all checks: %w", err)
	}
	defer rows.Close()

	var checks []Check
	for rows.Next() {
		var c Check
		var enabledInt int
		if err := rows.Scan(
			&c.ID, &c.DeviceID, &c.CheckType, &c.Target, &c.IntervalSeconds,
			&enabledInt, &c.CreatedAt, &c.UpdatedAt, &c.DeviceName,
		); err != nil {
			return nil, fmt.Errorf("scan check row: %w", err)
		}
		c.Enabled = enabledInt != 0
		checks = append(checks, c)
	}
	return checks, rows.Err()
}

// UpdateCheck updates a check's type, target, interval, and enabled state.
func (s *PulseStore) UpdateCheck(ctx context.Context, c *Check) error {
	enabledInt := 0
	if c.Enabled {
		enabledInt = 1
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE pulse_checks SET check_type = ?, target = ?, interval_seconds = ?, enabled = ?, updated_at = ?
		WHERE id = ?`,
		c.CheckType, c.Target, c.IntervalSeconds, enabledInt, c.UpdatedAt, c.ID,
	)
	if err != nil {
		return fmt.Errorf("update check: %w", err)
	}
	return nil
}

// DeleteCheck deletes a check and cascade-deletes its results.
func (s *PulseStore) DeleteCheck(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pulse_check_results WHERE check_id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete check results: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM pulse_checks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete check: %w", err)
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

// UpdateDeviceLastSeen updates the last_seen timestamp on a device after a successful check.
func (s *PulseStore) UpdateDeviceLastSeen(ctx context.Context, deviceID string, t time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE recon_devices SET last_seen = ? WHERE id = ?`,
		t.UTC().Format(time.RFC3339), deviceID,
	)
	if err != nil {
		return fmt.Errorf("update device last_seen: %w", err)
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

// -- Metrics Queries --

// validMetrics is the set of supported metric names for QueryMetrics.
var validMetrics = map[string]bool{
	"latency":      true,
	"packet_loss":  true,
	"success_rate": true,
}

// validRanges maps time range strings to their Go durations.
var validRanges = map[string]time.Duration{
	"1h":  1 * time.Hour,
	"6h":  6 * time.Hour,
	"24h": 24 * time.Hour,
	"7d":  7 * 24 * time.Hour,
	"30d": 30 * 24 * time.Hour,
}

// metricBucket accumulates values for a single time bucket during aggregation.
type metricBucket struct {
	latencySum    float64
	packetLossSum float64
	successCount  int
	total         int
}

// QueryMetrics returns aggregated time-series data for a device, with
// automatic downsampling based on the requested time range.
// Bucketing is performed in Go to avoid SQLite date-format parsing issues.
func (s *PulseStore) QueryMetrics(ctx context.Context, deviceID, metric, timeRange string) (*MetricSeries, error) {
	if !validMetrics[metric] {
		return nil, fmt.Errorf("unknown metric %q: must be latency, packet_loss, or success_rate", metric)
	}

	duration, ok := validRanges[timeRange]
	if !ok {
		return nil, fmt.Errorf("unknown range %q: must be 1h, 6h, 24h, 7d, or 30d", timeRange)
	}

	since := time.Now().UTC().Add(-duration)

	// Determine bucket size (seconds) based on range.
	var bucketSec int64
	switch {
	case duration <= 24*time.Hour:
		bucketSec = 60 // 1-minute buckets
	case duration <= 7*24*time.Hour:
		bucketSec = 300 // 5-minute buckets
	default:
		bucketSec = 3600 // 1-hour buckets
	}

	// Fetch raw results within the time range.
	rows, err := s.db.QueryContext(ctx, `
		SELECT latency_ms, packet_loss, success, checked_at
		FROM pulse_check_results
		WHERE device_id = ? AND checked_at >= ?
		ORDER BY checked_at ASC`,
		deviceID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("query metrics: %w", err)
	}
	defer rows.Close()

	// Aggregate results into time buckets in Go.
	buckets := make(map[int64]*metricBucket)
	var bucketKeys []int64
	for rows.Next() {
		var latency, packetLoss float64
		var successInt int
		var checkedAt time.Time
		if err := rows.Scan(&latency, &packetLoss, &successInt, &checkedAt); err != nil {
			return nil, fmt.Errorf("scan metric row: %w", err)
		}
		key := (checkedAt.Unix() / bucketSec) * bucketSec
		b, exists := buckets[key]
		if !exists {
			b = &metricBucket{}
			buckets[key] = b
			bucketKeys = append(bucketKeys, key)
		}
		b.latencySum += latency
		b.packetLossSum += packetLoss
		b.successCount += successInt
		b.total++
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric rows: %w", err)
	}

	// Convert buckets to data points (already ordered by checkedAt ASC).
	points := make([]MetricDataPoint, 0, len(bucketKeys))
	for _, key := range bucketKeys {
		b := buckets[key]
		var value float64
		switch metric {
		case "latency":
			value = b.latencySum / float64(b.total)
		case "packet_loss":
			value = b.packetLossSum / float64(b.total)
		case "success_rate":
			value = float64(b.successCount) * 100.0 / float64(b.total)
		}
		points = append(points, MetricDataPoint{
			Timestamp: time.Unix(key, 0).UTC(),
			Value:     value,
		})
	}

	return &MetricSeries{
		DeviceID: deviceID,
		Metric:   metric,
		Range:    timeRange,
		Points:   points,
	}, nil
}

// -- Alerts --

// InsertAlert inserts a new monitoring alert.
func (s *PulseStore) InsertAlert(ctx context.Context, a *Alert) error {
	var resolvedAt sql.NullTime
	if a.ResolvedAt != nil {
		resolvedAt = sql.NullTime{Time: *a.ResolvedAt, Valid: true}
	}
	suppressed := 0
	if a.Suppressed {
		suppressed = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pulse_alerts (
			id, check_id, device_id, severity, message, triggered_at, resolved_at,
			consecutive_failures, suppressed, suppressed_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.CheckID, a.DeviceID, a.Severity, a.Message,
		a.TriggeredAt, resolvedAt, a.ConsecutiveFailures,
		suppressed, a.SuppressedBy,
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
	var resolvedAt, acknowledgedAt sql.NullTime
	var suppressedInt int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, check_id, device_id, severity, message, triggered_at, resolved_at,
			acknowledged_at, consecutive_failures, suppressed, suppressed_by
		FROM pulse_alerts WHERE check_id = ? AND resolved_at IS NULL`,
		checkID,
	).Scan(
		&a.ID, &a.CheckID, &a.DeviceID, &a.Severity, &a.Message,
		&a.TriggeredAt, &resolvedAt, &acknowledgedAt, &a.ConsecutiveFailures,
		&suppressedInt, &a.SuppressedBy,
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
	if acknowledgedAt.Valid {
		a.AcknowledgedAt = &acknowledgedAt.Time
	}
	a.Suppressed = suppressedInt != 0
	return &a, nil
}

// ListActiveAlerts returns active (unresolved) alerts. If deviceID is empty, returns all
// active alerts. If deviceID is provided, returns only active alerts for that device.
// Device names are resolved via LEFT JOIN with recon_devices.
func (s *PulseStore) ListActiveAlerts(ctx context.Context, deviceID string) ([]Alert, error) {
	var rows *sql.Rows
	var err error
	if deviceID == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT a.id, a.check_id, a.device_id, a.severity, a.message, a.triggered_at, a.resolved_at,
				a.acknowledged_at, a.consecutive_failures, a.suppressed, a.suppressed_by,
				COALESCE(NULLIF(d.hostname, ''), json_extract(d.ip_addresses, '$[0]'), a.device_id) AS device_name
			FROM pulse_alerts a
			LEFT JOIN recon_devices d ON d.id = a.device_id
			WHERE a.resolved_at IS NULL ORDER BY a.triggered_at DESC`,
		)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT a.id, a.check_id, a.device_id, a.severity, a.message, a.triggered_at, a.resolved_at,
				a.acknowledged_at, a.consecutive_failures, a.suppressed, a.suppressed_by,
				COALESCE(NULLIF(d.hostname, ''), json_extract(d.ip_addresses, '$[0]'), a.device_id) AS device_name
			FROM pulse_alerts a
			LEFT JOIN recon_devices d ON d.id = a.device_id
			WHERE a.device_id = ? AND a.resolved_at IS NULL ORDER BY a.triggered_at DESC`,
			deviceID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("list active alerts: %w", err)
	}
	defer rows.Close()

	return scanAlertRows(rows)
}

// GetAlert returns a single alert by ID. Returns nil, nil if not found.
func (s *PulseStore) GetAlert(ctx context.Context, id string) (*Alert, error) {
	var a Alert
	var resolvedAt, acknowledgedAt sql.NullTime
	var suppressedInt int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, check_id, device_id, severity, message, triggered_at, resolved_at,
			acknowledged_at, consecutive_failures, suppressed, suppressed_by
		FROM pulse_alerts WHERE id = ?`,
		id,
	).Scan(
		&a.ID, &a.CheckID, &a.DeviceID, &a.Severity, &a.Message,
		&a.TriggeredAt, &resolvedAt, &acknowledgedAt, &a.ConsecutiveFailures,
		&suppressedInt, &a.SuppressedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get alert: %w", err)
	}
	if resolvedAt.Valid {
		a.ResolvedAt = &resolvedAt.Time
	}
	if acknowledgedAt.Valid {
		a.AcknowledgedAt = &acknowledgedAt.Time
	}
	a.Suppressed = suppressedInt != 0
	return &a, nil
}

// ListAlerts returns alerts matching the given filters.
// Device names are resolved via LEFT JOIN with recon_devices.
func (s *PulseStore) ListAlerts(ctx context.Context, filters AlertFilters) ([]Alert, error) {
	query := `SELECT a.id, a.check_id, a.device_id, a.severity, a.message, a.triggered_at, a.resolved_at,
		a.acknowledged_at, a.consecutive_failures, a.suppressed, a.suppressed_by,
		COALESCE(NULLIF(d.hostname, ''), json_extract(d.ip_addresses, '$[0]'), a.device_id) AS device_name
		FROM pulse_alerts a
		LEFT JOIN recon_devices d ON d.id = a.device_id`
	var conditions []string
	var args []any

	if filters.DeviceID != "" {
		conditions = append(conditions, "a.device_id = ?")
		args = append(args, filters.DeviceID)
	}
	if filters.Severity != "" {
		conditions = append(conditions, "a.severity = ?")
		args = append(args, filters.Severity)
	}
	if filters.ActiveOnly {
		conditions = append(conditions, "a.resolved_at IS NULL")
	}
	if filters.Suppressed != nil {
		if *filters.Suppressed {
			conditions = append(conditions, "a.suppressed = 1")
		} else {
			conditions = append(conditions, "a.suppressed = 0")
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY a.triggered_at DESC"

	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	return scanAlertRows(rows)
}

// AcknowledgeAlert sets the acknowledged_at timestamp on an alert.
func (s *PulseStore) AcknowledgeAlert(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE pulse_alerts SET acknowledged_at = ? WHERE id = ?`,
		time.Now().UTC(), id,
	)
	if err != nil {
		return fmt.Errorf("acknowledge alert: %w", err)
	}
	return nil
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

// scanAlertRows scans alert rows into a slice, handling nullable columns.
// Expects 12 columns: the standard 11 alert columns plus device_name.
func scanAlertRows(rows *sql.Rows) ([]Alert, error) {
	var alerts []Alert
	for rows.Next() {
		var a Alert
		var resolvedAt, acknowledgedAt sql.NullTime
		var suppressedInt int
		if err := rows.Scan(
			&a.ID, &a.CheckID, &a.DeviceID, &a.Severity, &a.Message,
			&a.TriggeredAt, &resolvedAt, &acknowledgedAt, &a.ConsecutiveFailures,
			&suppressedInt, &a.SuppressedBy, &a.DeviceName,
		); err != nil {
			return nil, fmt.Errorf("scan alert row: %w", err)
		}
		if resolvedAt.Valid {
			a.ResolvedAt = &resolvedAt.Time
		}
		if acknowledgedAt.Valid {
			a.AcknowledgedAt = &acknowledgedAt.Time
		}
		a.Suppressed = suppressedInt != 0
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

// -- Check Dependencies --

// AddCheckDependency creates a dependency between a check and an upstream device.
func (s *PulseStore) AddCheckDependency(ctx context.Context, checkID, dependsOnDeviceID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO pulse_check_dependencies (check_id, depends_on_device_id)
		VALUES (?, ?)`,
		checkID, dependsOnDeviceID,
	)
	if err != nil {
		return fmt.Errorf("add check dependency: %w", err)
	}
	return nil
}

// RemoveCheckDependency removes a dependency between a check and an upstream device.
func (s *PulseStore) RemoveCheckDependency(ctx context.Context, checkID, dependsOnDeviceID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM pulse_check_dependencies WHERE check_id = ? AND depends_on_device_id = ?`,
		checkID, dependsOnDeviceID,
	)
	if err != nil {
		return fmt.Errorf("remove check dependency: %w", err)
	}
	return nil
}

// ListCheckDependencies returns all dependencies for a check.
func (s *PulseStore) ListCheckDependencies(ctx context.Context, checkID string) ([]CheckDependency, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT check_id, depends_on_device_id, created_at
		FROM pulse_check_dependencies WHERE check_id = ? ORDER BY created_at`,
		checkID,
	)
	if err != nil {
		return nil, fmt.Errorf("list check dependencies: %w", err)
	}
	defer rows.Close()

	var deps []CheckDependency
	for rows.Next() {
		var d CheckDependency
		if err := rows.Scan(&d.CheckID, &d.DependsOnDeviceID, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan check dependency row: %w", err)
		}
		deps = append(deps, d)
	}
	return deps, rows.Err()
}

// GetDependentCheckIDs returns check IDs that depend on the given device.
func (s *PulseStore) GetDependentCheckIDs(ctx context.Context, deviceID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT check_id FROM pulse_check_dependencies WHERE depends_on_device_id = ?`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("get dependent check IDs: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan dependent check ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// IsSuppressed checks whether a check should be suppressed because an upstream device
// has an active critical alert. Returns the suppression state and the device ID causing it.
func (s *PulseStore) IsSuppressed(ctx context.Context, checkID string) (suppressed bool, byDevice string, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT d.depends_on_device_id
		FROM pulse_check_dependencies d
		JOIN pulse_alerts a ON a.device_id = d.depends_on_device_id
		WHERE d.check_id = ? AND a.resolved_at IS NULL AND a.severity = 'critical'
		LIMIT 1`,
		checkID,
	).Scan(&byDevice)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", fmt.Errorf("check suppression: %w", err)
	}
	return true, byDevice, nil
}

// -- Correlation Queries --

// GetParentActiveAlerts returns active alerts for the parent device of the given device,
// triggered within the specified time window. It joins against recon_devices to find the
// parent_device_id. Returns the matching alerts and the parent device ID.
func (s *PulseStore) GetParentActiveAlerts(ctx context.Context, deviceID string, window time.Duration) (alerts []Alert, parentDeviceID string, err error) {
	// First, resolve the parent device ID.
	err = s.db.QueryRowContext(ctx,
		`SELECT parent_device_id FROM recon_devices WHERE id = ? AND parent_device_id != ''`,
		deviceID,
	).Scan(&parentDeviceID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("get parent device: %w", err)
	}
	if parentDeviceID == "" {
		return nil, "", nil
	}

	// Query active alerts on the parent within the correlation window.
	since := time.Now().UTC().Add(-window)
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.check_id, a.device_id, a.severity, a.message, a.triggered_at, a.resolved_at,
			a.acknowledged_at, a.consecutive_failures, a.suppressed, a.suppressed_by,
			COALESCE(NULLIF(d.hostname, ''), json_extract(d.ip_addresses, '$[0]'), a.device_id) AS device_name
		FROM pulse_alerts a
		LEFT JOIN recon_devices d ON d.id = a.device_id
		WHERE a.device_id = ? AND a.resolved_at IS NULL AND a.triggered_at >= ?
		ORDER BY a.triggered_at DESC`,
		parentDeviceID, since,
	)
	if err != nil {
		return nil, parentDeviceID, fmt.Errorf("get parent active alerts: %w", err)
	}
	defer rows.Close()

	alerts, err = scanAlertRows(rows)
	if err != nil {
		return nil, parentDeviceID, err
	}
	return alerts, parentDeviceID, nil
}

// GetCorrelatedAlerts returns alerts grouped by correlation: parent alerts with
// their suppressed children. Only active (unresolved) alerts are included.
func (s *PulseStore) GetCorrelatedAlerts(ctx context.Context) ([]CorrelatedAlertGroup, error) {
	// Get all active non-suppressed alerts (potential parents).
	parentAlerts, err := s.ListAlerts(ctx, AlertFilters{
		ActiveOnly: true,
		Suppressed: boolPtr(false),
		Limit:      200,
	})
	if err != nil {
		return nil, fmt.Errorf("list parent alerts: %w", err)
	}

	// Get all active suppressed alerts.
	suppressedAlerts, err := s.ListAlerts(ctx, AlertFilters{
		ActiveOnly: true,
		Suppressed: boolPtr(true),
		Limit:      500,
	})
	if err != nil {
		return nil, fmt.Errorf("list suppressed alerts: %w", err)
	}

	// Build a map from device_id -> parent alert for grouping.
	deviceAlertMap := make(map[string]int, len(parentAlerts))
	groups := make([]CorrelatedAlertGroup, 0, len(parentAlerts))
	for i := range parentAlerts {
		deviceAlertMap[parentAlerts[i].DeviceID] = i
		groups = append(groups, CorrelatedAlertGroup{
			ParentAlert:        parentAlerts[i],
			SuppressedChildren: []Alert{},
		})
	}

	// Assign suppressed alerts to their parent groups.
	var ungrouped []Alert
	for i := range suppressedAlerts {
		parentIdx, ok := deviceAlertMap[suppressedAlerts[i].SuppressedBy]
		if ok {
			groups[parentIdx].SuppressedChildren = append(
				groups[parentIdx].SuppressedChildren,
				suppressedAlerts[i],
			)
		} else {
			ungrouped = append(ungrouped, suppressedAlerts[i])
		}
	}

	// Add ungrouped suppressed alerts as standalone groups (parent resolved but child still active).
	for i := range ungrouped {
		groups = append(groups, CorrelatedAlertGroup{
			ParentAlert:        ungrouped[i],
			SuppressedChildren: []Alert{},
		})
	}

	return groups, nil
}

// boolPtr returns a pointer to a bool value.
func boolPtr(v bool) *bool {
	return &v
}

// -- Notification Channels --

// InsertChannel inserts a new notification channel.
func (s *PulseStore) InsertChannel(ctx context.Context, ch *NotificationChannel) error {
	enabled := 0
	if ch.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pulse_notification_channels (id, name, type, config, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ch.ID, ch.Name, ch.Type, ch.Config, enabled, ch.CreatedAt, ch.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert notification channel: %w", err)
	}
	return nil
}

// GetChannel returns a notification channel by ID. Returns nil, nil if not found.
func (s *PulseStore) GetChannel(ctx context.Context, id string) (*NotificationChannel, error) {
	var ch NotificationChannel
	var enabledInt int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, type, config, enabled, created_at, updated_at
		FROM pulse_notification_channels WHERE id = ?`,
		id,
	).Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &enabledInt, &ch.CreatedAt, &ch.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get notification channel: %w", err)
	}
	ch.Enabled = enabledInt != 0
	return &ch, nil
}

// ListChannels returns all notification channels.
func (s *PulseStore) ListChannels(ctx context.Context) ([]NotificationChannel, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, type, config, enabled, created_at, updated_at
		FROM pulse_notification_channels ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("list notification channels: %w", err)
	}
	defer rows.Close()

	var channels []NotificationChannel
	for rows.Next() {
		var ch NotificationChannel
		var enabledInt int
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &enabledInt, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan notification channel row: %w", err)
		}
		ch.Enabled = enabledInt != 0
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

// ListEnabledChannels returns only enabled notification channels.
func (s *PulseStore) ListEnabledChannels(ctx context.Context) ([]NotificationChannel, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, type, config, enabled, created_at, updated_at
		FROM pulse_notification_channels WHERE enabled = 1 ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled notification channels: %w", err)
	}
	defer rows.Close()

	var channels []NotificationChannel
	for rows.Next() {
		var ch NotificationChannel
		var enabledInt int
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &enabledInt, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan notification channel row: %w", err)
		}
		ch.Enabled = enabledInt != 0
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

// UpdateChannel updates a notification channel.
func (s *PulseStore) UpdateChannel(ctx context.Context, ch *NotificationChannel) error {
	enabled := 0
	if ch.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE pulse_notification_channels SET name = ?, config = ?, enabled = ?, updated_at = ?
		WHERE id = ?`,
		ch.Name, ch.Config, enabled, ch.UpdatedAt, ch.ID,
	)
	if err != nil {
		return fmt.Errorf("update notification channel: %w", err)
	}
	return nil
}

// DeleteChannel deletes a notification channel by ID.
func (s *PulseStore) DeleteChannel(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pulse_notification_channels WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete notification channel: %w", err)
	}
	return nil
}
