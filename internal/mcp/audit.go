package mcp

import (
	"context"
	"database/sql"
	"time"
)

// AuditEntry represents a single MCP tool invocation record.
type AuditEntry struct {
	ID           int64     `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	ToolName     string    `json:"tool_name"`
	InputJSON    string    `json:"input_json"`
	UserID       string    `json:"user_id"`
	DurationMs   int64     `json:"duration_ms"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// AuditStore handles persistence for MCP tool call audit records.
type AuditStore struct {
	db *sql.DB
}

// NewAuditStore creates a new AuditStore backed by the given database.
// The caller is responsible for running migrations via deps.Store.Migrate before
// passing deps.Store.DB() here.
func NewAuditStore(db *sql.DB) *AuditStore {
	return &AuditStore{db: db}
}

// Insert records an audit entry.
func (s *AuditStore) Insert(ctx context.Context, entry AuditEntry) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO mcp_audit_log (timestamp, tool_name, input_json, user_id, duration_ms, success, error_message)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.Timestamp.UTC().Format(time.RFC3339Nano),
		entry.ToolName,
		entry.InputJSON,
		entry.UserID,
		entry.DurationMs,
		boolToInt(entry.Success),
		entry.ErrorMessage,
	)
	return err
}

// List returns audit entries with optional filtering by tool name.
// Returns entries ordered by timestamp descending, total row count, and any error.
func (s *AuditStore) List(ctx context.Context, toolName string, limit, offset int) ([]AuditEntry, int, error) {
	// Count query.
	countQuery := "SELECT COUNT(*) FROM mcp_audit_log"
	var filterArgs []any
	if toolName != "" {
		countQuery += " WHERE tool_name = ?"
		filterArgs = append(filterArgs, toolName)
	}

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, filterArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query -- same WHERE clause plus pagination parameters.
	query := "SELECT id, timestamp, tool_name, input_json, user_id, duration_ms, success, error_message FROM mcp_audit_log"
	if toolName != "" {
		query += " WHERE tool_name = ?"
	}
	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	dataArgs := make([]any, 0, len(filterArgs)+2)
	dataArgs = append(dataArgs, filterArgs...)
	dataArgs = append(dataArgs, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries := make([]AuditEntry, 0, limit)
	for rows.Next() {
		var e AuditEntry
		var ts string
		var success int
		if err := rows.Scan(&e.ID, &ts, &e.ToolName, &e.InputJSON, &e.UserID, &e.DurationMs, &success, &e.ErrorMessage); err != nil {
			return nil, 0, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		e.Success = success != 0
		entries = append(entries, e)
	}

	return entries, total, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
