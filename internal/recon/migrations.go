package recon

import (
	"database/sql"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

// migrations returns the Recon module's database migrations.
func migrations() []plugin.Migration {
	return []plugin.Migration{
		{
			Version:     1,
			Description: "create recon tables (devices, scans, scan_devices, topology_links)",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE recon_devices (
						id               TEXT PRIMARY KEY,
						hostname         TEXT NOT NULL DEFAULT '',
						ip_addresses     TEXT NOT NULL DEFAULT '[]',
						mac_address      TEXT NOT NULL DEFAULT '',
						manufacturer     TEXT NOT NULL DEFAULT '',
						device_type      TEXT NOT NULL DEFAULT 'unknown',
						os               TEXT NOT NULL DEFAULT '',
						status           TEXT NOT NULL DEFAULT 'unknown',
						discovery_method TEXT NOT NULL DEFAULT 'icmp',
						agent_id         TEXT NOT NULL DEFAULT '',
						first_seen       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						last_seen        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						notes            TEXT NOT NULL DEFAULT '',
						tags             TEXT NOT NULL DEFAULT '[]',
						custom_fields    TEXT NOT NULL DEFAULT '{}'
					)`,
					`CREATE INDEX idx_recon_devices_mac ON recon_devices(mac_address)`,
					`CREATE INDEX idx_recon_devices_status ON recon_devices(status)`,
					`CREATE INDEX idx_recon_devices_last_seen ON recon_devices(last_seen)`,
					`CREATE TABLE recon_scans (
						id         TEXT PRIMARY KEY,
						subnet     TEXT NOT NULL,
						started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						ended_at   DATETIME,
						status     TEXT NOT NULL DEFAULT 'pending',
						total      INTEGER NOT NULL DEFAULT 0,
						online     INTEGER NOT NULL DEFAULT 0,
						error_msg  TEXT NOT NULL DEFAULT ''
					)`,
					`CREATE INDEX idx_recon_scans_status ON recon_scans(status)`,
					`CREATE TABLE recon_scan_devices (
						scan_id   TEXT NOT NULL REFERENCES recon_scans(id) ON DELETE CASCADE,
						device_id TEXT NOT NULL REFERENCES recon_devices(id) ON DELETE CASCADE,
						PRIMARY KEY (scan_id, device_id)
					)`,
					`CREATE TABLE recon_topology_links (
						id               TEXT PRIMARY KEY,
						source_device_id TEXT NOT NULL REFERENCES recon_devices(id) ON DELETE CASCADE,
						target_device_id TEXT NOT NULL REFERENCES recon_devices(id) ON DELETE CASCADE,
						source_port      TEXT NOT NULL DEFAULT '',
						target_port      TEXT NOT NULL DEFAULT '',
						link_type        TEXT NOT NULL DEFAULT 'arp',
						speed            INTEGER NOT NULL DEFAULT 0,
						discovered_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						last_confirmed   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
					)`,
					`CREATE UNIQUE INDEX idx_recon_topology_pair ON recon_topology_links(source_device_id, target_device_id, link_type)`,
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
			Description: "create recon_device_history table for status change tracking",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE IF NOT EXISTS recon_device_history (
						id TEXT PRIMARY KEY,
						device_id TEXT NOT NULL,
						old_status TEXT NOT NULL,
						new_status TEXT NOT NULL,
						changed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
						FOREIGN KEY (device_id) REFERENCES recon_devices(id) ON DELETE CASCADE
					)`,
					`CREATE INDEX IF NOT EXISTS idx_recon_device_history_device ON recon_device_history(device_id)`,
					`CREATE INDEX IF NOT EXISTS idx_recon_device_history_changed ON recon_device_history(changed_at)`,
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
			Version:     3,
			Description: "add inventory fields (location, category, primary_role, owner) to recon_devices",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`ALTER TABLE recon_devices ADD COLUMN location TEXT NOT NULL DEFAULT ''`,
					`ALTER TABLE recon_devices ADD COLUMN category TEXT NOT NULL DEFAULT ''`,
					`ALTER TABLE recon_devices ADD COLUMN primary_role TEXT NOT NULL DEFAULT ''`,
					`ALTER TABLE recon_devices ADD COLUMN owner TEXT NOT NULL DEFAULT ''`,
					`CREATE INDEX idx_recon_devices_category ON recon_devices(category)`,
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
			Version:     4,
			Description: "create topology_layouts table for server-side layout persistence",
			Up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS recon_topology_layouts (
					id TEXT PRIMARY KEY,
					name TEXT NOT NULL,
					positions TEXT NOT NULL DEFAULT '[]',
					created_at TEXT NOT NULL DEFAULT (datetime('now')),
					updated_at TEXT NOT NULL DEFAULT (datetime('now'))
				)`)
				return err
			},
		},
	}
}
