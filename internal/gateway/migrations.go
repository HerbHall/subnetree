package gateway

import (
	"database/sql"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

func migrations() []plugin.Migration {
	return []plugin.Migration{
		{
			Version:     1,
			Description: "create gateway audit log table",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE IF NOT EXISTS gateway_audit_log (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						session_id TEXT NOT NULL,
						device_id TEXT NOT NULL,
						user_id TEXT NOT NULL DEFAULT '',
						session_type TEXT NOT NULL,
						target TEXT NOT NULL,
						action TEXT NOT NULL,
						bytes_in INTEGER DEFAULT 0,
						bytes_out INTEGER DEFAULT 0,
						source_ip TEXT DEFAULT '',
						timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`,
					`CREATE INDEX IF NOT EXISTS idx_gateway_audit_timestamp ON gateway_audit_log(timestamp)`,
					`CREATE INDEX IF NOT EXISTS idx_gateway_audit_device ON gateway_audit_log(device_id)`,
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
