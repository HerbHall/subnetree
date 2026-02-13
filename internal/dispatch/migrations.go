package dispatch

import (
	"context"
	"database/sql"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

func migrations() []plugin.Migration {
	return []plugin.Migration{
		{
			Version:     1,
			Description: "create dispatch tables for agent management and enrollment",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE IF NOT EXISTS dispatch_agents (
						id TEXT PRIMARY KEY,
						hostname TEXT NOT NULL DEFAULT '',
						platform TEXT NOT NULL DEFAULT '',
						agent_version TEXT NOT NULL DEFAULT '',
						proto_version INTEGER NOT NULL DEFAULT 1,
						device_id TEXT NOT NULL DEFAULT '',
						status TEXT NOT NULL DEFAULT 'pending',
						last_check_in DATETIME,
						enrolled_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						cert_serial TEXT NOT NULL DEFAULT '',
						cert_expires_at DATETIME,
						config_json TEXT NOT NULL DEFAULT '{}'
					)`,
					`CREATE INDEX IF NOT EXISTS idx_dispatch_agents_status ON dispatch_agents(status)`,
					`CREATE INDEX IF NOT EXISTS idx_dispatch_agents_device ON dispatch_agents(device_id)`,
					`CREATE TABLE IF NOT EXISTS dispatch_enrollment_tokens (
						id TEXT PRIMARY KEY,
						token_hash TEXT NOT NULL UNIQUE,
						description TEXT NOT NULL DEFAULT '',
						created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						expires_at DATETIME,
						used_at DATETIME,
						agent_id TEXT,
						max_uses INTEGER NOT NULL DEFAULT 1,
						use_count INTEGER NOT NULL DEFAULT 0
					)`,
					`CREATE INDEX IF NOT EXISTS idx_dispatch_tokens_hash ON dispatch_enrollment_tokens(token_hash)`,
				}
				for _, stmt := range stmts {
					if _, err := tx.ExecContext(context.Background(), stmt); err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			Version:     2,
			Description: "create dispatch device profiles table for system profiling",
			Up: func(tx *sql.Tx) error {
				_, err := tx.ExecContext(context.Background(), `
					CREATE TABLE IF NOT EXISTS dispatch_device_profiles (
						agent_id TEXT PRIMARY KEY REFERENCES dispatch_agents(id) ON DELETE CASCADE,
						hardware_json TEXT NOT NULL DEFAULT '{}',
						software_json TEXT NOT NULL DEFAULT '{}',
						services_json TEXT NOT NULL DEFAULT '[]',
						collected_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`)
				return err
			},
		},
	}
}
