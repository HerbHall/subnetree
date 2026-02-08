package vault

import (
	"database/sql"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

func migrations() []plugin.Migration {
	return []plugin.Migration{
		{
			Version:     1,
			Description: "create vault credential tables",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE IF NOT EXISTS vault_master (
						id INTEGER PRIMARY KEY CHECK (id = 1),
						salt BLOB NOT NULL,
						verification_blob BLOB NOT NULL,
						created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`,

					`CREATE TABLE IF NOT EXISTS vault_credentials (
						id TEXT PRIMARY KEY,
						name TEXT NOT NULL,
						type TEXT NOT NULL,
						device_id TEXT,
						description TEXT DEFAULT '',
						encrypted_data BLOB NOT NULL,
						created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`,
					`CREATE INDEX IF NOT EXISTS idx_vault_credentials_device ON vault_credentials(device_id)`,
					`CREATE INDEX IF NOT EXISTS idx_vault_credentials_type ON vault_credentials(type)`,

					`CREATE TABLE IF NOT EXISTS vault_keys (
						credential_id TEXT PRIMARY KEY REFERENCES vault_credentials(id) ON DELETE CASCADE,
						wrapped_key BLOB NOT NULL,
						created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`,

					`CREATE TABLE IF NOT EXISTS vault_audit_log (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						credential_id TEXT NOT NULL,
						user_id TEXT NOT NULL DEFAULT '',
						action TEXT NOT NULL,
						purpose TEXT DEFAULT '',
						source_ip TEXT DEFAULT '',
						timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`,
					`CREATE INDEX IF NOT EXISTS idx_vault_audit_timestamp ON vault_audit_log(timestamp)`,
					`CREATE INDEX IF NOT EXISTS idx_vault_audit_credential ON vault_audit_log(credential_id)`,
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
