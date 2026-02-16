package autodoc

import (
	"database/sql"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

// migrations returns the AutoDoc module's database migrations.
func migrations() []plugin.Migration {
	return []plugin.Migration{
		{
			Version:     1,
			Description: "create autodoc_changelog table",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE IF NOT EXISTS autodoc_changelog (
						id            TEXT PRIMARY KEY,
						event_type    TEXT NOT NULL,
						summary       TEXT NOT NULL,
						details       TEXT NOT NULL DEFAULT '{}',
						source_module TEXT NOT NULL DEFAULT '',
						device_id     TEXT,
						created_at    TEXT NOT NULL
					)`,
					`CREATE INDEX IF NOT EXISTS idx_autodoc_created_at ON autodoc_changelog(created_at)`,
					`CREATE INDEX IF NOT EXISTS idx_autodoc_event_type ON autodoc_changelog(event_type)`,
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
