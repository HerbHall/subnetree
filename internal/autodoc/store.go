package autodoc

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Store provides database operations for the AutoDoc module.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store backed by the given database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ChangelogEntry represents a single auto-documentation changelog entry.
type ChangelogEntry struct {
	ID           string          `json:"id"`
	EventType    string          `json:"event_type"`
	Summary      string          `json:"summary"`
	Details      json.RawMessage `json:"details"`
	SourceModule string          `json:"source_module"`
	DeviceID     *string         `json:"device_id,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// ListFilter controls pagination and filtering for changelog queries.
type ListFilter struct {
	Page      int
	PerPage   int
	EventType string
	Since     *time.Time
	Until     *time.Time
}

// ExportOptions controls the markdown export parameters.
type ExportOptions struct {
	Format string
	Since  time.Time
	Until  time.Time
}

// Stats provides aggregate statistics about the changelog.
type Stats struct {
	TotalEntries  int              `json:"total_entries"`
	EntriesByType map[string]int   `json:"entries_by_type"`
	LatestEntry   *ChangelogEntry  `json:"latest_entry,omitempty"`
	OldestEntry   *ChangelogEntry  `json:"oldest_entry,omitempty"`
}

// SaveEntry inserts a new changelog entry.
func (s *Store) SaveEntry(ctx context.Context, entry ChangelogEntry) error {
	details := entry.Details
	if details == nil {
		details = json.RawMessage("{}")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO autodoc_changelog (id, event_type, summary, details, source_module, device_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.ID, entry.EventType, entry.Summary, string(details),
		entry.SourceModule, entry.DeviceID,
		entry.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert changelog entry: %w", err)
	}
	return nil
}

// ListEntries returns a paginated list of changelog entries with optional filters.
func (s *Store) ListEntries(ctx context.Context, filter ListFilter) ([]ChangelogEntry, int, error) {
	if filter.PerPage <= 0 {
		filter.PerPage = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.PerPage

	where, args := s.buildWhere(filter)

	// Count total matching entries.
	var total int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM autodoc_changelog WHERE "+where, args..., //nolint:gosec // where uses parameterized placeholders only
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count changelog entries: %w", err)
	}

	// Query with pagination.
	queryArgs := make([]any, 0, len(args)+2)
	queryArgs = append(queryArgs, args...)
	queryArgs = append(queryArgs, filter.PerPage, offset)

	//nolint:gosec // where uses parameterized placeholders only
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, event_type, summary, details, source_module, device_id, created_at "+
			"FROM autodoc_changelog WHERE "+where+
			" ORDER BY created_at DESC LIMIT ? OFFSET ?",
		queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list changelog entries: %w", err)
	}
	defer rows.Close()

	entries := make([]ChangelogEntry, 0)
	for rows.Next() {
		entry, scanErr := s.scanEntry(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		entries = append(entries, *entry)
	}
	return entries, total, rows.Err()
}

// GetStats returns aggregate statistics about the changelog.
func (s *Store) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{
		EntriesByType: make(map[string]int),
	}

	// Total count.
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM autodoc_changelog`,
	).Scan(&stats.TotalEntries)
	if err != nil {
		return nil, fmt.Errorf("count total: %w", err)
	}

	// Group by event_type.
	typeRows, err := s.db.QueryContext(ctx,
		`SELECT event_type, COUNT(*) FROM autodoc_changelog GROUP BY event_type`)
	if err != nil {
		return nil, fmt.Errorf("count by type: %w", err)
	}
	defer typeRows.Close()
	for typeRows.Next() {
		var et string
		var cnt int
		if err := typeRows.Scan(&et, &cnt); err != nil {
			return nil, fmt.Errorf("scan type row: %w", err)
		}
		stats.EntriesByType[et] = cnt
	}
	if err := typeRows.Err(); err != nil {
		return nil, fmt.Errorf("type rows: %w", err)
	}

	// Latest entry.
	latestRow := s.db.QueryRowContext(ctx,
		`SELECT id, event_type, summary, details, source_module, device_id, created_at
		 FROM autodoc_changelog ORDER BY created_at DESC LIMIT 1`)
	latest, err := s.scanEntryRow(latestRow)
	if err == nil {
		stats.LatestEntry = latest
	}

	// Oldest entry.
	oldestRow := s.db.QueryRowContext(ctx,
		`SELECT id, event_type, summary, details, source_module, device_id, created_at
		 FROM autodoc_changelog ORDER BY created_at ASC LIMIT 1`)
	oldest, err := s.scanEntryRow(oldestRow)
	if err == nil {
		stats.OldestEntry = oldest
	}

	return stats, nil
}

// ListEntriesSince returns all entries created after the given time, ordered chronologically.
func (s *Store) ListEntriesSince(ctx context.Context, since time.Time) ([]ChangelogEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, event_type, summary, details, source_module, device_id, created_at
		 FROM autodoc_changelog
		 WHERE created_at >= ?
		 ORDER BY created_at ASC`,
		since.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("list entries since: %w", err)
	}
	defer rows.Close()

	var entries []ChangelogEntry
	for rows.Next() {
		entry, scanErr := s.scanEntry(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, *entry)
	}
	return entries, rows.Err()
}

// ListEntriesBetween returns all entries in the given time range, ordered chronologically.
func (s *Store) ListEntriesBetween(ctx context.Context, since, until time.Time) ([]ChangelogEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, event_type, summary, details, source_module, device_id, created_at
		 FROM autodoc_changelog
		 WHERE created_at >= ? AND created_at <= ?
		 ORDER BY created_at ASC`,
		since.UTC().Format(time.RFC3339),
		until.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("list entries between: %w", err)
	}
	defer rows.Close()

	var entries []ChangelogEntry
	for rows.Next() {
		entry, scanErr := s.scanEntry(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		entries = append(entries, *entry)
	}
	return entries, rows.Err()
}

// buildWhere constructs a WHERE clause from the filter.
func (s *Store) buildWhere(filter ListFilter) (string, []any) {
	clauses := []string{"1=1"}
	args := []any{}

	if filter.EventType != "" {
		clauses = append(clauses, "event_type = ?")
		args = append(args, filter.EventType)
	}
	if filter.Since != nil {
		clauses = append(clauses, "created_at >= ?")
		args = append(args, filter.Since.UTC().Format(time.RFC3339))
	}
	if filter.Until != nil {
		clauses = append(clauses, "created_at <= ?")
		args = append(args, filter.Until.UTC().Format(time.RFC3339))
	}

	return strings.Join(clauses, " AND "), args
}

// scanEntry scans a *sql.Rows row into a ChangelogEntry.
func (s *Store) scanEntry(rows *sql.Rows) (*ChangelogEntry, error) {
	var entry ChangelogEntry
	var details string
	var createdAt string
	var deviceID sql.NullString

	err := rows.Scan(
		&entry.ID, &entry.EventType, &entry.Summary, &details,
		&entry.SourceModule, &deviceID, &createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan changelog entry: %w", err)
	}

	entry.Details = json.RawMessage(details)
	if deviceID.Valid {
		entry.DeviceID = &deviceID.String
	}
	entry.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &entry, nil
}

// scanEntryRow scans a *sql.Row into a ChangelogEntry.
func (s *Store) scanEntryRow(row *sql.Row) (*ChangelogEntry, error) {
	var entry ChangelogEntry
	var details string
	var createdAt string
	var deviceID sql.NullString

	err := row.Scan(
		&entry.ID, &entry.EventType, &entry.Summary, &details,
		&entry.SourceModule, &deviceID, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	entry.Details = json.RawMessage(details)
	if deviceID.Valid {
		entry.DeviceID = &deviceID.String
	}
	entry.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &entry, nil
}
