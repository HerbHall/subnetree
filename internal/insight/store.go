package insight

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/pkg/analytics"
)

// InsightStore provides database access for the Insight analytics plugin.
type InsightStore struct {
	db *sql.DB
}

// NewInsightStore creates a new InsightStore backed by the given database.
func NewInsightStore(db *sql.DB) *InsightStore {
	return &InsightStore{db: db}
}

// -- Baselines --

// UpsertBaseline inserts or updates a baseline record.
func (s *InsightStore) UpsertBaseline(ctx context.Context, b *analytics.Baseline) error {
	stable := 0
	if b.Stable {
		stable = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO analytics_baselines (
			device_id, metric_name, algorithm, mean, std_dev, variance,
			samples, stable, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.DeviceID, b.MetricName, b.Algorithm, b.Mean, b.StdDev,
		b.StdDev*b.StdDev, b.Samples, stable, b.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert baseline: %w", err)
	}
	return nil
}

// GetBaselines returns all baselines for a device.
func (s *InsightStore) GetBaselines(ctx context.Context, deviceID string) ([]analytics.Baseline, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT device_id, metric_name, algorithm, mean, std_dev, samples, stable, updated_at
		FROM analytics_baselines WHERE device_id = ? ORDER BY metric_name`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("get baselines: %w", err)
	}
	defer rows.Close()

	var baselines []analytics.Baseline
	for rows.Next() {
		var b analytics.Baseline
		var stableInt int
		if err := rows.Scan(
			&b.DeviceID, &b.MetricName, &b.Algorithm,
			&b.Mean, &b.StdDev, &b.Samples, &stableInt, &b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan baseline row: %w", err)
		}
		b.Stable = stableInt != 0
		baselines = append(baselines, b)
	}
	return baselines, rows.Err()
}

// -- Anomalies --

// InsertAnomaly inserts a new anomaly record.
func (s *InsightStore) InsertAnomaly(ctx context.Context, a *analytics.Anomaly) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO analytics_anomalies (
			id, device_id, metric_name, severity, type,
			value, expected, deviation, description, detected_at, resolved_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.DeviceID, a.MetricName, a.Severity, a.Type,
		a.Value, a.Expected, a.Deviation, a.Description, a.DetectedAt, a.ResolvedAt,
	)
	if err != nil {
		return fmt.Errorf("insert anomaly: %w", err)
	}
	return nil
}

// ListAnomalies returns anomalies, optionally filtered by device.
// Pass empty deviceID to list all. Results are ordered by detected_at descending.
func (s *InsightStore) ListAnomalies(ctx context.Context, deviceID string, limit int) ([]analytics.Anomaly, error) {
	if limit <= 0 {
		limit = 50
	}

	var rows *sql.Rows
	var err error
	if deviceID == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, device_id, metric_name, severity, type,
				value, expected, deviation, description, detected_at, resolved_at
			FROM analytics_anomalies ORDER BY detected_at DESC LIMIT ?`,
			limit,
		)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, device_id, metric_name, severity, type,
				value, expected, deviation, description, detected_at, resolved_at
			FROM analytics_anomalies WHERE device_id = ? ORDER BY detected_at DESC LIMIT ?`,
			deviceID, limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("list anomalies: %w", err)
	}
	defer rows.Close()

	var anomalies []analytics.Anomaly
	for rows.Next() {
		var a analytics.Anomaly
		var resolvedAt sql.NullTime
		if err := rows.Scan(
			&a.ID, &a.DeviceID, &a.MetricName, &a.Severity, &a.Type,
			&a.Value, &a.Expected, &a.Deviation, &a.Description,
			&a.DetectedAt, &resolvedAt,
		); err != nil {
			return nil, fmt.Errorf("scan anomaly row: %w", err)
		}
		if resolvedAt.Valid {
			a.ResolvedAt = &resolvedAt.Time
		}
		anomalies = append(anomalies, a)
	}
	return anomalies, rows.Err()
}

// ResolveAnomaly marks an anomaly as resolved.
func (s *InsightStore) ResolveAnomaly(ctx context.Context, id string, resolvedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE analytics_anomalies SET resolved_at = ? WHERE id = ?`,
		resolvedAt, id,
	)
	if err != nil {
		return fmt.Errorf("resolve anomaly: %w", err)
	}
	return nil
}

// DeleteOldAnomalies deletes resolved anomalies older than the given time.
// Returns the number of rows deleted.
func (s *InsightStore) DeleteOldAnomalies(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM analytics_anomalies WHERE resolved_at IS NOT NULL AND resolved_at < ?`,
		before,
	)
	if err != nil {
		return 0, fmt.Errorf("delete old anomalies: %w", err)
	}
	return result.RowsAffected()
}

// -- Forecasts --

// UpsertForecast inserts or updates a forecast record.
func (s *InsightStore) UpsertForecast(ctx context.Context, f *analytics.Forecast) error {
	var thresholdSecs *int64
	if f.TimeToThreshold != nil {
		secs := int64(f.TimeToThreshold.Seconds())
		thresholdSecs = &secs
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO analytics_forecasts (
			device_id, metric_name, current_value, predicted_value,
			threshold, slope, confidence, time_to_threshold_secs, generated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		f.DeviceID, f.MetricName, f.CurrentValue, f.PredictedValue,
		f.Threshold, f.Slope, f.Confidence, thresholdSecs, f.GeneratedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert forecast: %w", err)
	}
	return nil
}

// GetForecasts returns all forecasts for a device.
func (s *InsightStore) GetForecasts(ctx context.Context, deviceID string) ([]analytics.Forecast, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT device_id, metric_name, current_value, predicted_value,
			threshold, slope, confidence, time_to_threshold_secs, generated_at
		FROM analytics_forecasts WHERE device_id = ?`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("get forecasts: %w", err)
	}
	defer rows.Close()

	var forecasts []analytics.Forecast
	for rows.Next() {
		var f analytics.Forecast
		var thresholdSecs sql.NullInt64
		if err := rows.Scan(
			&f.DeviceID, &f.MetricName, &f.CurrentValue, &f.PredictedValue,
			&f.Threshold, &f.Slope, &f.Confidence, &thresholdSecs, &f.GeneratedAt,
		); err != nil {
			return nil, fmt.Errorf("scan forecast row: %w", err)
		}
		if thresholdSecs.Valid {
			d := time.Duration(thresholdSecs.Int64) * time.Second
			f.TimeToThreshold = &d
		}
		forecasts = append(forecasts, f)
	}
	return forecasts, rows.Err()
}

// -- Correlations --

// InsertCorrelation inserts a new alert correlation group.
func (s *InsightStore) InsertCorrelation(ctx context.Context, g *analytics.AlertGroup) error {
	deviceIDsJSON, err := json.Marshal(g.DeviceIDs)
	if err != nil {
		return fmt.Errorf("marshal device_ids: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO analytics_correlations (
			id, root_cause, device_ids, alert_count, description, created_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
		g.ID, g.RootCause, string(deviceIDsJSON), g.AlertCount, g.Description, g.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert correlation: %w", err)
	}
	return nil
}

// ListActiveCorrelations returns correlation groups that have not been resolved.
func (s *InsightStore) ListActiveCorrelations(ctx context.Context) ([]analytics.AlertGroup, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, root_cause, device_ids, alert_count, description, created_at
		FROM analytics_correlations WHERE resolved_at IS NULL ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list active correlations: %w", err)
	}
	defer rows.Close()

	var groups []analytics.AlertGroup
	for rows.Next() {
		var g analytics.AlertGroup
		var deviceIDsJSON string
		if err := rows.Scan(
			&g.ID, &g.RootCause, &deviceIDsJSON, &g.AlertCount,
			&g.Description, &g.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan correlation row: %w", err)
		}
		if err := json.Unmarshal([]byte(deviceIDsJSON), &g.DeviceIDs); err != nil {
			return nil, fmt.Errorf("unmarshal device_ids: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

// -- Metrics --

// InsertMetric stores a raw metric data point.
func (s *InsightStore) InsertMetric(ctx context.Context, p *analytics.MetricPoint) error {
	tagsJSON, err := json.Marshal(p.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO analytics_metrics (device_id, metric_name, value, tags, timestamp)
		VALUES (?, ?, ?, ?, ?)`,
		p.DeviceID, p.MetricName, p.Value, string(tagsJSON), p.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert metric: %w", err)
	}
	return nil
}

// GetMetricWindow returns metric points for a device+metric within a time window.
func (s *InsightStore) GetMetricWindow(ctx context.Context, deviceID, metricName string, since time.Time) ([]analytics.MetricPoint, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT device_id, metric_name, value, tags, timestamp
		FROM analytics_metrics
		WHERE device_id = ? AND metric_name = ? AND timestamp >= ?
		ORDER BY timestamp`,
		deviceID, metricName, since,
	)
	if err != nil {
		return nil, fmt.Errorf("get metric window: %w", err)
	}
	defer rows.Close()

	var points []analytics.MetricPoint
	for rows.Next() {
		var p analytics.MetricPoint
		var tagsJSON string
		if err := rows.Scan(&p.DeviceID, &p.MetricName, &p.Value, &tagsJSON, &p.Timestamp); err != nil {
			return nil, fmt.Errorf("scan metric row: %w", err)
		}
		if err := json.Unmarshal([]byte(tagsJSON), &p.Tags); err != nil {
			return nil, fmt.Errorf("unmarshal tags: %w", err)
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// GetLatestMetricPerDevice returns the most recent metric point per device for a given metric name.
func (s *InsightStore) GetLatestMetricPerDevice(ctx context.Context, metricName string, since time.Time) ([]analytics.MetricPoint, error) {
	query := `
		SELECT device_id, metric_name, value, timestamp
		FROM analytics_metrics
		WHERE metric_name = ? AND timestamp >= ?
		GROUP BY device_id
		HAVING timestamp = MAX(timestamp)
		ORDER BY value DESC`

	rows, err := s.db.QueryContext(ctx, query, metricName, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("query latest metrics: %w", err)
	}
	defer rows.Close()

	var results []analytics.MetricPoint
	for rows.Next() {
		var p analytics.MetricPoint
		var ts string
		if err := rows.Scan(&p.DeviceID, &p.MetricName, &p.Value, &ts); err != nil {
			return nil, fmt.Errorf("scan metric: %w", err)
		}
		p.Timestamp, _ = time.Parse(time.RFC3339, ts)
		results = append(results, p)
	}
	return results, rows.Err()
}

// DeleteOldMetrics deletes metrics older than the given time.
// Returns the number of rows deleted.
func (s *InsightStore) DeleteOldMetrics(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM analytics_metrics WHERE timestamp < ?`,
		before,
	)
	if err != nil {
		return 0, fmt.Errorf("delete old metrics: %w", err)
	}
	return result.RowsAffected()
}
