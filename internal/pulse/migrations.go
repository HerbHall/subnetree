package pulse

import (
	"database/sql"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

func migrations() []plugin.Migration {
	return []plugin.Migration{
		{
			Version:     1,
			Description: "create pulse monitoring tables",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE IF NOT EXISTS pulse_checks (
						id TEXT PRIMARY KEY,
						device_id TEXT NOT NULL,
						check_type TEXT NOT NULL DEFAULT 'icmp',
						target TEXT NOT NULL,
						interval_seconds INTEGER NOT NULL DEFAULT 30,
						enabled INTEGER NOT NULL DEFAULT 1,
						created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`,
					`CREATE INDEX IF NOT EXISTS idx_pulse_checks_device ON pulse_checks(device_id)`,

					`CREATE TABLE IF NOT EXISTS pulse_check_results (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						check_id TEXT NOT NULL REFERENCES pulse_checks(id),
						device_id TEXT NOT NULL,
						success INTEGER NOT NULL,
						latency_ms REAL,
						packet_loss REAL,
						error_message TEXT,
						checked_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`,
					`CREATE INDEX IF NOT EXISTS idx_pulse_results_device_time ON pulse_check_results(device_id, checked_at)`,
					`CREATE INDEX IF NOT EXISTS idx_pulse_results_check_time ON pulse_check_results(check_id, checked_at)`,

					`CREATE TABLE IF NOT EXISTS pulse_alerts (
						id TEXT PRIMARY KEY,
						check_id TEXT NOT NULL REFERENCES pulse_checks(id),
						device_id TEXT NOT NULL,
						severity TEXT NOT NULL DEFAULT 'warning',
						message TEXT NOT NULL,
						triggered_at DATETIME NOT NULL,
						resolved_at DATETIME,
						consecutive_failures INTEGER NOT NULL DEFAULT 0
					)`,
					`CREATE INDEX IF NOT EXISTS idx_pulse_alerts_device ON pulse_alerts(device_id, resolved_at)`,
				}
				for _, stmt := range stmts {
					if _, err := tx.Exec(stmt); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			Version:     2,
			Description: "add acknowledged_at to pulse_alerts",
			Up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`ALTER TABLE pulse_alerts ADD COLUMN acknowledged_at DATETIME`)
				return err
			},
		},
		{
			Version:     3,
			Description: "create notification channels table",
			Up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS pulse_notification_channels (
					id TEXT PRIMARY KEY,
					name TEXT NOT NULL,
					type TEXT NOT NULL,
					config TEXT NOT NULL,
					enabled INTEGER NOT NULL DEFAULT 1,
					created_at DATETIME NOT NULL,
					updated_at DATETIME NOT NULL
				)`)
				return err
			},
		},
	}
}
