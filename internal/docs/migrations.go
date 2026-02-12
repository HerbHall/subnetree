package docs

import (
	"database/sql"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

func migrations() []plugin.Migration {
	return []plugin.Migration{
		{
			Version:     1,
			Description: "create docs tables (applications, snapshots)",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE IF NOT EXISTS docs_applications (
						id            TEXT PRIMARY KEY,
						name          TEXT NOT NULL,
						app_type      TEXT NOT NULL DEFAULT 'unknown',
						device_id     TEXT NOT NULL DEFAULT '',
						collector     TEXT NOT NULL DEFAULT '',
						status        TEXT NOT NULL DEFAULT 'active',
						metadata      TEXT NOT NULL DEFAULT '{}',
						discovered_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`,
					`CREATE INDEX IF NOT EXISTS idx_docs_applications_type ON docs_applications(app_type)`,
					`CREATE INDEX IF NOT EXISTS idx_docs_applications_device ON docs_applications(device_id)`,
					`CREATE INDEX IF NOT EXISTS idx_docs_applications_status ON docs_applications(status)`,

					`CREATE TABLE IF NOT EXISTS docs_snapshots (
						id             TEXT PRIMARY KEY,
						application_id TEXT NOT NULL REFERENCES docs_applications(id) ON DELETE CASCADE,
						content_hash   TEXT NOT NULL,
						content        TEXT NOT NULL,
						format         TEXT NOT NULL DEFAULT 'json',
						size_bytes     INTEGER NOT NULL DEFAULT 0,
						source         TEXT NOT NULL DEFAULT 'auto',
						captured_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`,
					`CREATE INDEX IF NOT EXISTS idx_docs_snapshots_app ON docs_snapshots(application_id, captured_at)`,
					`CREATE INDEX IF NOT EXISTS idx_docs_snapshots_hash ON docs_snapshots(content_hash)`,
					`CREATE INDEX IF NOT EXISTS idx_docs_snapshots_captured ON docs_snapshots(captured_at)`,
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
