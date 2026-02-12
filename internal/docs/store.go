package docs

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Application represents a discovered infrastructure application.
type Application struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	AppType      string    `json:"app_type"`
	DeviceID     string    `json:"device_id,omitempty"`
	Collector    string    `json:"collector"`
	Status       string    `json:"status"`
	Metadata     string    `json:"metadata"`
	DiscoveredAt time.Time `json:"discovered_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Snapshot represents a point-in-time capture of an application's configuration.
type Snapshot struct {
	ID            string    `json:"id"`
	ApplicationID string    `json:"application_id"`
	ContentHash   string    `json:"content_hash"`
	Content       string    `json:"content"`
	Format        string    `json:"format"`
	SizeBytes     int       `json:"size_bytes"`
	Source        string    `json:"source"`
	CapturedAt    time.Time `json:"captured_at"`
}

// ListApplicationsParams controls filtering and pagination for application queries.
type ListApplicationsParams struct {
	Limit   int
	Offset  int
	AppType string
	Status  string
}

// ListSnapshotsParams controls filtering and pagination for snapshot queries.
type ListSnapshotsParams struct {
	ApplicationID string
	Limit         int
	Offset        int
}

// DocsStore provides database access for the Docs module.
type DocsStore struct {
	db *sql.DB
}

// NewStore creates a new DocsStore backed by the given database.
func NewStore(db *sql.DB) *DocsStore {
	return &DocsStore{db: db}
}

// -- Applications --

// InsertApplication inserts a new application record.
func (s *DocsStore) InsertApplication(ctx context.Context, a *Application) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO docs_applications (
			id, name, app_type, device_id, collector, status, metadata, discovered_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Name, a.AppType, a.DeviceID, a.Collector, a.Status, a.Metadata,
		a.DiscoveredAt, a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert application: %w", err)
	}
	return nil
}

// GetApplication returns an application by ID. Returns nil, nil if not found.
func (s *DocsStore) GetApplication(ctx context.Context, id string) (*Application, error) {
	var a Application
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, app_type, device_id, collector, status, metadata, discovered_at, updated_at
		FROM docs_applications WHERE id = ?`,
		id,
	).Scan(
		&a.ID, &a.Name, &a.AppType, &a.DeviceID, &a.Collector, &a.Status,
		&a.Metadata, &a.DiscoveredAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get application: %w", err)
	}
	return &a, nil
}

// ListApplications returns a filtered, paginated list of applications with total count.
func (s *DocsStore) ListApplications(ctx context.Context, params ListApplicationsParams) ([]Application, int, error) {
	if params.Limit <= 0 {
		params.Limit = 50
	}

	where := "1=1"
	args := []any{}
	if params.AppType != "" {
		where += " AND app_type = ?"
		args = append(args, params.AppType)
	}
	if params.Status != "" {
		where += " AND status = ?"
		args = append(args, params.Status)
	}

	var total int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM docs_applications WHERE "+where, args..., //nolint:gosec // where uses parameterized placeholders only
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count applications: %w", err)
	}

	queryArgs := make([]any, 0, len(args)+2)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, params.Limit, params.Offset)
	//nolint:gosec // where uses parameterized placeholders only
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, app_type, device_id, collector, status, metadata, discovered_at, updated_at "+
			"FROM docs_applications WHERE "+where+" ORDER BY updated_at DESC LIMIT ? OFFSET ?",
		queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var apps []Application
	for rows.Next() {
		var a Application
		if err := rows.Scan(
			&a.ID, &a.Name, &a.AppType, &a.DeviceID, &a.Collector, &a.Status,
			&a.Metadata, &a.DiscoveredAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan application row: %w", err)
		}
		apps = append(apps, a)
	}
	return apps, total, rows.Err()
}

// UpsertApplication inserts a new application or updates it if the ID already exists.
func (s *DocsStore) UpsertApplication(ctx context.Context, a *Application) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO docs_applications (
			id, name, app_type, device_id, collector, status, metadata, discovered_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			app_type = excluded.app_type,
			device_id = excluded.device_id,
			collector = excluded.collector,
			status = excluded.status,
			metadata = excluded.metadata,
			updated_at = excluded.updated_at`,
		a.ID, a.Name, a.AppType, a.DeviceID, a.Collector, a.Status, a.Metadata,
		a.DiscoveredAt, a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert application: %w", err)
	}
	return nil
}

// UpdateApplication updates an existing application record.
func (s *DocsStore) UpdateApplication(ctx context.Context, a *Application) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE docs_applications SET
			name = ?, app_type = ?, device_id = ?, collector = ?, status = ?,
			metadata = ?, updated_at = ?
		WHERE id = ?`,
		a.Name, a.AppType, a.DeviceID, a.Collector, a.Status,
		a.Metadata, a.UpdatedAt, a.ID,
	)
	if err != nil {
		return fmt.Errorf("update application: %w", err)
	}
	return nil
}

// -- Snapshots --

// InsertSnapshot inserts a new snapshot record.
func (s *DocsStore) InsertSnapshot(ctx context.Context, snap *Snapshot) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO docs_snapshots (
			id, application_id, content_hash, content, format, size_bytes, source, captured_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		snap.ID, snap.ApplicationID, snap.ContentHash, snap.Content, snap.Format,
		snap.SizeBytes, snap.Source, snap.CapturedAt,
	)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	return nil
}

// GetSnapshot returns a snapshot by ID. Returns nil, nil if not found.
func (s *DocsStore) GetSnapshot(ctx context.Context, id string) (*Snapshot, error) {
	var snap Snapshot
	err := s.db.QueryRowContext(ctx, `
		SELECT id, application_id, content_hash, content, format, size_bytes, source, captured_at
		FROM docs_snapshots WHERE id = ?`,
		id,
	).Scan(
		&snap.ID, &snap.ApplicationID, &snap.ContentHash, &snap.Content,
		&snap.Format, &snap.SizeBytes, &snap.Source, &snap.CapturedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	return &snap, nil
}

// ListSnapshots returns a filtered, paginated list of snapshots.
func (s *DocsStore) ListSnapshots(ctx context.Context, params ListSnapshotsParams) ([]Snapshot, error) {
	if params.Limit <= 0 {
		params.Limit = 50
	}

	where := "1=1"
	args := []any{}
	if params.ApplicationID != "" {
		where += " AND application_id = ?"
		args = append(args, params.ApplicationID)
	}

	queryArgs := make([]any, 0, len(args)+2)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, params.Limit, params.Offset)
	//nolint:gosec // where uses parameterized placeholders only
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, application_id, content_hash, content, format, size_bytes, source, captured_at "+
			"FROM docs_snapshots WHERE "+where+" ORDER BY captured_at DESC LIMIT ? OFFSET ?",
		queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []Snapshot
	for rows.Next() {
		var snap Snapshot
		if err := rows.Scan(
			&snap.ID, &snap.ApplicationID, &snap.ContentHash, &snap.Content,
			&snap.Format, &snap.SizeBytes, &snap.Source, &snap.CapturedAt,
		); err != nil {
			return nil, fmt.Errorf("scan snapshot row: %w", err)
		}
		snapshots = append(snapshots, snap)
	}
	return snapshots, rows.Err()
}

// GetLatestSnapshot returns the most recent snapshot for an application. Returns nil, nil if none.
func (s *DocsStore) GetLatestSnapshot(ctx context.Context, applicationID string) (*Snapshot, error) {
	var snap Snapshot
	err := s.db.QueryRowContext(ctx, `
		SELECT id, application_id, content_hash, content, format, size_bytes, source, captured_at
		FROM docs_snapshots WHERE application_id = ? ORDER BY captured_at DESC LIMIT 1`,
		applicationID,
	).Scan(
		&snap.ID, &snap.ApplicationID, &snap.ContentHash, &snap.Content,
		&snap.Format, &snap.SizeBytes, &snap.Source, &snap.CapturedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest snapshot: %w", err)
	}
	return &snap, nil
}

// DeleteSnapshot deletes a snapshot by ID.
func (s *DocsStore) DeleteSnapshot(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM docs_snapshots WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete snapshot: %w", err)
	}
	return nil
}

// CountSnapshots returns the number of snapshots for an application.
func (s *DocsStore) CountSnapshots(ctx context.Context, applicationID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM docs_snapshots WHERE application_id = ?`,
		applicationID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count snapshots: %w", err)
	}
	return count, nil
}

// DeleteOldSnapshots deletes snapshots captured before the given time.
// Returns the number of rows deleted.
func (s *DocsStore) DeleteOldSnapshots(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM docs_snapshots WHERE captured_at < ?`,
		before,
	)
	if err != nil {
		return 0, fmt.Errorf("delete old snapshots: %w", err)
	}
	return result.RowsAffected()
}

// ApplicationStats holds aggregate statistics for an application's snapshots.
type ApplicationStats struct {
	SnapshotCount  int       `json:"snapshot_count"`
	LatestCapture  time.Time `json:"latest_capture"`
	TotalSizeBytes int64     `json:"total_size_bytes"`
}

// ListSnapshotHistory returns paginated snapshots for an application ordered by
// capture time descending, along with the total count.
func (s *DocsStore) ListSnapshotHistory(ctx context.Context, applicationID string, limit, offset int) ([]Snapshot, int, error) {
	if limit <= 0 {
		limit = 20
	}

	var total int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM docs_snapshots WHERE application_id = ?`,
		applicationID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count snapshot history: %w", err)
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, application_id, content_hash, content, format, size_bytes, source, captured_at
		 FROM docs_snapshots WHERE application_id = ?
		 ORDER BY captured_at DESC LIMIT ? OFFSET ?`,
		applicationID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list snapshot history: %w", err)
	}
	defer rows.Close()

	var snapshots []Snapshot
	for rows.Next() {
		var snap Snapshot
		if err := rows.Scan(
			&snap.ID, &snap.ApplicationID, &snap.ContentHash, &snap.Content,
			&snap.Format, &snap.SizeBytes, &snap.Source, &snap.CapturedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan snapshot history row: %w", err)
		}
		snapshots = append(snapshots, snap)
	}
	return snapshots, total, rows.Err()
}

// DeleteExcessSnapshots keeps only the most recent maxCount snapshots for an
// application, deleting any older ones. Returns the number deleted.
func (s *DocsStore) DeleteExcessSnapshots(ctx context.Context, applicationID string, maxCount int) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM docs_snapshots
		WHERE application_id = ?
		  AND id NOT IN (
			SELECT id FROM docs_snapshots
			WHERE application_id = ?
			ORDER BY captured_at DESC
			LIMIT ?
		  )`,
		applicationID, applicationID, maxCount,
	)
	if err != nil {
		return 0, fmt.Errorf("delete excess snapshots: %w", err)
	}
	return result.RowsAffected()
}

// GetApplicationStats returns aggregate snapshot statistics for an application.
func (s *DocsStore) GetApplicationStats(ctx context.Context, id string) (*ApplicationStats, error) {
	var stats ApplicationStats
	var latestCapture sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(MAX(captured_at), '0001-01-01'), COALESCE(SUM(size_bytes), 0)
		FROM docs_snapshots WHERE application_id = ?`,
		id,
	).Scan(&stats.SnapshotCount, &latestCapture, &stats.TotalSizeBytes)
	if err != nil {
		return nil, fmt.Errorf("get application stats: %w", err)
	}
	if latestCapture.Valid {
		stats.LatestCapture = latestCapture.Time
	}
	return &stats, nil
}
