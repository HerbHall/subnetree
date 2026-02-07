package insight

import (
	"database/sql"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

// migrations returns the Insight module's database migrations.
func migrations() []plugin.Migration {
	return []plugin.Migration{
		{
			Version:     1,
			Description: "create analytics tables",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE IF NOT EXISTS analytics_baselines (
						device_id    TEXT NOT NULL,
						metric_name  TEXT NOT NULL,
						algorithm    TEXT NOT NULL DEFAULT 'ewma',
						mean         REAL NOT NULL DEFAULT 0,
						std_dev      REAL NOT NULL DEFAULT 0,
						variance     REAL NOT NULL DEFAULT 0,
						samples      INTEGER NOT NULL DEFAULT 0,
						stable       INTEGER NOT NULL DEFAULT 0,
						updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (device_id, metric_name)
					)`,
					`CREATE INDEX IF NOT EXISTS idx_analytics_baselines_device ON analytics_baselines(device_id)`,

					`CREATE TABLE IF NOT EXISTS analytics_anomalies (
						id           TEXT PRIMARY KEY,
						device_id    TEXT NOT NULL,
						metric_name  TEXT NOT NULL,
						severity     TEXT NOT NULL DEFAULT 'warning',
						type         TEXT NOT NULL DEFAULT 'zscore',
						value        REAL NOT NULL,
						expected     REAL NOT NULL,
						deviation    REAL NOT NULL,
						description  TEXT NOT NULL DEFAULT '',
						detected_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						resolved_at  DATETIME
					)`,
					`CREATE INDEX IF NOT EXISTS idx_analytics_anomalies_device ON analytics_anomalies(device_id)`,
					`CREATE INDEX IF NOT EXISTS idx_analytics_anomalies_detected ON analytics_anomalies(detected_at)`,

					`CREATE TABLE IF NOT EXISTS analytics_forecasts (
						device_id       TEXT NOT NULL,
						metric_name     TEXT NOT NULL,
						current_value   REAL NOT NULL DEFAULT 0,
						predicted_value REAL NOT NULL DEFAULT 0,
						threshold       REAL NOT NULL DEFAULT 0,
						slope           REAL NOT NULL DEFAULT 0,
						confidence      REAL NOT NULL DEFAULT 0,
						time_to_threshold_secs INTEGER,
						generated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (device_id, metric_name)
					)`,

					`CREATE TABLE IF NOT EXISTS analytics_correlations (
						id           TEXT PRIMARY KEY,
						root_cause   TEXT NOT NULL DEFAULT '',
						device_ids   TEXT NOT NULL DEFAULT '[]',
						alert_count  INTEGER NOT NULL DEFAULT 0,
						description  TEXT NOT NULL DEFAULT '',
						created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						resolved_at  DATETIME
					)`,
					`CREATE INDEX IF NOT EXISTS idx_analytics_correlations_created ON analytics_correlations(created_at)`,

					`CREATE TABLE IF NOT EXISTS analytics_metrics (
						id          INTEGER PRIMARY KEY AUTOINCREMENT,
						device_id   TEXT NOT NULL,
						metric_name TEXT NOT NULL,
						value       REAL NOT NULL,
						tags        TEXT NOT NULL DEFAULT '{}',
						timestamp   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`,
					`CREATE INDEX IF NOT EXISTS idx_analytics_metrics_device_time ON analytics_metrics(device_id, metric_name, timestamp)`,
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
