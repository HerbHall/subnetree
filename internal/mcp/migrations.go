package mcp

import (
	"database/sql"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

func migrations() []plugin.Migration {
	return []plugin.Migration{
		{
			Version:     1,
			Description: "create mcp audit log table",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE IF NOT EXISTS mcp_audit_log (
						id            INTEGER PRIMARY KEY AUTOINCREMENT,
						timestamp     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
						tool_name     TEXT    NOT NULL,
						input_json    TEXT    NOT NULL DEFAULT '{}',
						user_id       TEXT    NOT NULL DEFAULT 'stdio',
						duration_ms   INTEGER NOT NULL DEFAULT 0,
						success       INTEGER NOT NULL DEFAULT 1,
						error_message TEXT    NOT NULL DEFAULT ''
					)`,
					`CREATE INDEX IF NOT EXISTS idx_mcp_audit_timestamp ON mcp_audit_log(timestamp)`,
					`CREATE INDEX IF NOT EXISTS idx_mcp_audit_tool ON mcp_audit_log(tool_name)`,
				}
				for _, stmt := range stmts {
					if _, err := tx.Exec(stmt); err != nil {
						return err
					}
				}
				return nil
			},
		},
	}
}
