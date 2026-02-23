package recon

import (
	"database/sql"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

// Migrations returns the Recon module's database migrations.
// Exported for use by cross-package test infrastructure.
func Migrations() []plugin.Migration {
	return migrations()
}

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
		{
			Version:     5,
			Description: "create service_movements table for port migration tracking",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE IF NOT EXISTS recon_service_movements (
						id TEXT PRIMARY KEY,
						port INTEGER NOT NULL,
						protocol TEXT NOT NULL DEFAULT 'tcp',
						service_name TEXT NOT NULL DEFAULT '',
						from_device_id TEXT NOT NULL,
						to_device_id TEXT NOT NULL,
						detected_at TEXT NOT NULL
					)`,
					`CREATE INDEX IF NOT EXISTS idx_service_movements_detected_at ON recon_service_movements(detected_at)`,
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
			Version:     6,
			Description: "create scan_metrics table for per-scan timing and count data",
			Up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS recon_scan_metrics (
					scan_id TEXT PRIMARY KEY REFERENCES recon_scans(id) ON DELETE CASCADE,
					duration_ms INTEGER NOT NULL,
					ping_phase_ms INTEGER NOT NULL,
					enrich_phase_ms INTEGER NOT NULL,
					post_process_ms INTEGER NOT NULL,
					hosts_scanned INTEGER NOT NULL,
					hosts_alive INTEGER NOT NULL,
					devices_created INTEGER NOT NULL,
					devices_updated INTEGER NOT NULL,
					created_at TEXT NOT NULL DEFAULT (datetime('now'))
				)`)
				return err
			},
		},
		{
			Version:     7,
			Description: "create scan_metrics_aggregates table for weekly/monthly rollups",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE IF NOT EXISTS recon_scan_metrics_aggregates (
						id TEXT PRIMARY KEY,
						period TEXT NOT NULL,
						period_start TEXT NOT NULL,
						period_end TEXT NOT NULL,
						scan_count INTEGER NOT NULL,
						avg_duration_ms REAL,
						avg_ping_phase_ms REAL,
						avg_enrich_ms REAL,
						avg_devices_found REAL,
						max_devices_found INTEGER,
						min_devices_found INTEGER,
						avg_hosts_alive REAL,
						total_new_devices INTEGER,
						failed_scans INTEGER NOT NULL DEFAULT 0,
						created_at TEXT NOT NULL DEFAULT (datetime('now'))
					)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS idx_scan_agg_period ON recon_scan_metrics_aggregates(period, period_start)`,
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
			Version:     8,
			Description: "add classification confidence columns to recon_devices",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`ALTER TABLE recon_devices ADD COLUMN classification_confidence INTEGER NOT NULL DEFAULT 0`,
					`ALTER TABLE recon_devices ADD COLUMN classification_source TEXT NOT NULL DEFAULT ''`,
					`ALTER TABLE recon_devices ADD COLUMN classification_signals TEXT NOT NULL DEFAULT ''`,
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
			Version:     9,
			Description: "add network hierarchy columns to recon_devices",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`ALTER TABLE recon_devices ADD COLUMN parent_device_id TEXT NOT NULL DEFAULT ''`,
					`ALTER TABLE recon_devices ADD COLUMN network_layer INTEGER NOT NULL DEFAULT 0`,
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
			Version:     10,
			Description: "create hardware asset profile tables (hardware, storage, gpu, services)",
			Up: func(tx *sql.Tx) error {
				stmts := []string{
					`CREATE TABLE recon_device_hardware (
						device_id           TEXT PRIMARY KEY REFERENCES recon_devices(id) ON DELETE CASCADE,
						hostname            TEXT NOT NULL DEFAULT '',
						fqdn                TEXT NOT NULL DEFAULT '',
						os_name             TEXT NOT NULL DEFAULT '',
						os_version          TEXT NOT NULL DEFAULT '',
						os_arch             TEXT NOT NULL DEFAULT '',
						kernel              TEXT NOT NULL DEFAULT '',
						cpu_model           TEXT NOT NULL DEFAULT '',
						cpu_cores           INTEGER NOT NULL DEFAULT 0,
						cpu_threads         INTEGER NOT NULL DEFAULT 0,
						cpu_arch            TEXT NOT NULL DEFAULT '',
						ram_total_mb        INTEGER NOT NULL DEFAULT 0,
						ram_type            TEXT NOT NULL DEFAULT '',
						ram_slots_used      INTEGER NOT NULL DEFAULT 0,
						ram_slots_total     INTEGER NOT NULL DEFAULT 0,
						platform_type       TEXT NOT NULL DEFAULT '',
						hypervisor          TEXT NOT NULL DEFAULT '',
						vm_host_id          TEXT NOT NULL DEFAULT '',
						system_manufacturer TEXT NOT NULL DEFAULT '',
						system_model        TEXT NOT NULL DEFAULT '',
						serial_number       TEXT NOT NULL DEFAULT '',
						bios_version        TEXT NOT NULL DEFAULT '',
						collection_source   TEXT NOT NULL DEFAULT '',
						collected_at        DATETIME,
						updated_at          DATETIME
					)`,
					`CREATE INDEX idx_recon_device_hardware_cpu ON recon_device_hardware(cpu_model)`,
					`CREATE INDEX idx_recon_device_hardware_os ON recon_device_hardware(os_name)`,
					`CREATE INDEX idx_recon_device_hardware_platform ON recon_device_hardware(platform_type)`,
					`CREATE TABLE recon_device_storage (
						id                TEXT PRIMARY KEY,
						device_id         TEXT NOT NULL REFERENCES recon_devices(id) ON DELETE CASCADE,
						name              TEXT NOT NULL DEFAULT '',
						disk_type         TEXT NOT NULL DEFAULT '',
						interface         TEXT NOT NULL DEFAULT '',
						capacity_gb       INTEGER NOT NULL DEFAULT 0,
						model             TEXT NOT NULL DEFAULT '',
						role              TEXT NOT NULL DEFAULT '',
						collection_source TEXT NOT NULL DEFAULT '',
						collected_at      DATETIME
					)`,
					`CREATE INDEX idx_recon_device_storage_device ON recon_device_storage(device_id)`,
					`CREATE TABLE recon_device_gpu (
						id                TEXT PRIMARY KEY,
						device_id         TEXT NOT NULL REFERENCES recon_devices(id) ON DELETE CASCADE,
						model             TEXT NOT NULL DEFAULT '',
						vendor            TEXT NOT NULL DEFAULT '',
						vram_mb           INTEGER NOT NULL DEFAULT 0,
						driver_version    TEXT NOT NULL DEFAULT '',
						collection_source TEXT NOT NULL DEFAULT '',
						collected_at      DATETIME
					)`,
					`CREATE INDEX idx_recon_device_gpu_device ON recon_device_gpu(device_id)`,
					`CREATE TABLE recon_device_services (
						id                TEXT PRIMARY KEY,
						device_id         TEXT NOT NULL REFERENCES recon_devices(id) ON DELETE CASCADE,
						name              TEXT NOT NULL DEFAULT '',
						service_type      TEXT NOT NULL DEFAULT '',
						port              INTEGER NOT NULL DEFAULT 0,
						url               TEXT NOT NULL DEFAULT '',
						version           TEXT NOT NULL DEFAULT '',
						status            TEXT NOT NULL DEFAULT '',
						collection_source TEXT NOT NULL DEFAULT '',
						collected_at      DATETIME
					)`,
					`CREATE INDEX idx_recon_device_services_device ON recon_device_services(device_id)`,
					`CREATE INDEX idx_recon_device_services_name ON recon_device_services(name)`,
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
			Version:     11,
			Description: "add connection_type column to recon_devices for WiFi heuristic detection",
			Up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`ALTER TABLE recon_devices ADD COLUMN connection_type TEXT NOT NULL DEFAULT 'unknown'`)
				return err
			},
		},
		{
			Version:     12,
			Description: "create recon_proxmox_resources table for VM/container resource snapshots",
			Up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`CREATE TABLE recon_proxmox_resources (
					device_id     TEXT PRIMARY KEY REFERENCES recon_devices(id) ON DELETE CASCADE,
					cpu_percent   REAL NOT NULL DEFAULT 0,
					mem_used_mb   INTEGER NOT NULL DEFAULT 0,
					mem_total_mb  INTEGER NOT NULL DEFAULT 0,
					disk_used_gb  INTEGER NOT NULL DEFAULT 0,
					disk_total_gb INTEGER NOT NULL DEFAULT 0,
					uptime_sec    INTEGER NOT NULL DEFAULT 0,
					netin_bytes   INTEGER NOT NULL DEFAULT 0,
					netout_bytes  INTEGER NOT NULL DEFAULT 0,
					collected_at  DATETIME NOT NULL
				)`)
				return err
			},
		},
		{
			Version:     13,
			Description: "create recon_wifi_clients table for WiFi AP client snapshots",
			Up: func(tx *sql.Tx) error {
				_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS recon_wifi_clients (
					device_id      TEXT PRIMARY KEY REFERENCES recon_devices(id) ON DELETE CASCADE,
					signal_dbm     INTEGER NOT NULL DEFAULT 0,
					signal_avg_dbm INTEGER NOT NULL DEFAULT 0,
					connected_sec  INTEGER NOT NULL DEFAULT 0,
					inactive_sec   INTEGER NOT NULL DEFAULT 0,
					rx_bitrate_bps INTEGER NOT NULL DEFAULT 0,
					tx_bitrate_bps INTEGER NOT NULL DEFAULT 0,
					rx_bytes       INTEGER NOT NULL DEFAULT 0,
					tx_bytes       INTEGER NOT NULL DEFAULT 0,
					ap_bssid       TEXT NOT NULL DEFAULT '',
					ap_ssid        TEXT NOT NULL DEFAULT '',
					collected_at   DATETIME NOT NULL
				)`)
				return err
			},
		},
	}
}
