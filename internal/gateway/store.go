package gateway

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// GatewayStore provides database access for the Gateway module.
type GatewayStore struct {
	db *sql.DB
}

// NewGatewayStore creates a new GatewayStore wrapping the given database connection.
func NewGatewayStore(db *sql.DB) *GatewayStore {
	return &GatewayStore{db: db}
}

// InsertAuditEntry records a gateway access event.
func (s *GatewayStore) InsertAuditEntry(ctx context.Context, entry *AuditEntry) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO gateway_audit_log (session_id, device_id, user_id, session_type, target, action, bytes_in, bytes_out, source_ip, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.SessionID, entry.DeviceID, entry.UserID, entry.SessionType,
		entry.Target, entry.Action, entry.BytesIn, entry.BytesOut,
		entry.SourceIP, entry.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert gateway audit entry: %w", err)
	}
	return nil
}

// ListAuditEntries returns audit entries, optionally filtered by device ID.
// Pass empty deviceID to list all entries.
func (s *GatewayStore) ListAuditEntries(ctx context.Context, deviceID string, limit int) ([]AuditEntry, error) {
	var query string
	var args []any

	if deviceID != "" {
		query = `SELECT id, session_id, device_id, user_id, session_type, target, action, bytes_in, bytes_out, source_ip, timestamp
			FROM gateway_audit_log WHERE device_id = ? ORDER BY timestamp DESC LIMIT ?`
		args = []any{deviceID, limit}
	} else {
		query = `SELECT id, session_id, device_id, user_id, session_type, target, action, bytes_in, bytes_out, source_ip, timestamp
			FROM gateway_audit_log ORDER BY timestamp DESC LIMIT ?`
		args = []any{limit}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list gateway audit entries: %w", err)
	}
	defer rows.Close()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.SessionID, &e.DeviceID, &e.UserID,
			&e.SessionType, &e.Target, &e.Action, &e.BytesIn, &e.BytesOut,
			&e.SourceIP, &e.Timestamp); err != nil {
			return nil, fmt.Errorf("scan gateway audit row: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// DeleteOldAuditEntries deletes audit entries older than the given time.
func (s *GatewayStore) DeleteOldAuditEntries(ctx context.Context, before time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM gateway_audit_log WHERE timestamp < ?`, before)
	if err != nil {
		return 0, fmt.Errorf("delete old gateway audit entries: %w", err)
	}
	return result.RowsAffected()
}

// ListAuditEntriesByDevice returns audit entries for a specific device.
func (s *GatewayStore) ListAuditEntriesByDevice(ctx context.Context, deviceID string, limit int) ([]AuditEntry, error) {
	return s.ListAuditEntries(ctx, deviceID, limit)
}
