## Operations & Maintenance

### Backup & Restore

#### What to Back Up

| Component | Location | Method |
|-----------|----------|--------|
| Database | `data/netvantage.db` | SQLite online backup API (safe during operation) |
| Configuration | `config.yaml` + env vars | File copy |
| TLS certificates | `data/certs/` | File copy (CA key, server cert, agent certs) |
| OUI database | Embedded in binary | Not needed (re-embedded on upgrade) |
| Vault master key | Not on disk (derived from passphrase) | User must retain passphrase |

#### Backup Commands

```bash
netvantage backup --output /path/to/backup.tar.gz    # Full backup (DB + config + certs)
netvantage restore --input /path/to/backup.tar.gz     # Restore to current data dir
netvantage backup --db-only --output /path/to/db.bak  # Database-only backup
```

- Online backup: safe to run while server is operating (uses SQLite backup API)
- Restore to different host: supported (for disaster recovery / migration)
- Automated backups: configurable schedule in `config.yaml` with retention count

#### Backup Configuration

```yaml
backup:
  enabled: false
  schedule: "0 2 * * *"      # Cron expression (daily at 2 AM)
  retention_count: 7          # Keep last N backups
  output_dir: "./data/backups"
```

### Data Retention

Configurable per data type with automated purge. Defaults balance storage with useful history.

| Data Type | Default Retention | Configurable | Purge Method |
|-----------|------------------|--------------|--------------|
| Raw device metrics | 7 days | Yes | Automated daily purge |
| Scan results | 30 days | Yes | Automated daily purge |
| Alerts / events | 180 days | Yes | Automated daily purge |
| Audit logs | 1 year | Yes | Automated daily purge |
| Agent check-in records | 7 days | Yes | Automated daily purge |
| Aggregated metrics (Phase 2) | 1 year | Yes | TimescaleDB retention policy |
| Device records | Never (manual delete) | No | User-initiated |

Configuration:

```yaml
retention:
  metrics_days: 7
  scans_days: 30
  alerts_days: 180
  audit_days: 365
  checkins_days: 7
  purge_schedule: "0 3 * * *"  # Daily at 3 AM
```

### Database Maintenance

- **SQLite WAL checkpointing:** Automatic on server shutdown; configurable periodic checkpoint during operation
- **SQLite VACUUM:** Manual via CLI command `netvantage db vacuum`; not automatic (can be slow on large databases)
- **Database size monitoring:** Exposed as Prometheus metric `netvantage_db_size_bytes`

### Upgrade Strategy

#### Server Upgrades

- Replace binary + restart. Database schema migrations run automatically on startup.
- Migrations are forward-only (no automatic rollback). Take a backup before upgrading.
- Server logs applied migrations at startup for auditability.
- Upgrade path: any version within the same major version can upgrade directly to the latest. Major version upgrades may require intermediate steps (documented in release notes).

#### Agent-Server Version Compatibility

See **Scout Agent Specification > Agent-Server Version Compatibility** for the full compatibility table and version negotiation protocol.

**Summary rule:** Agents must be the same or older proto version as the server (server supports current and N-1). Always upgrade the server first, then agents. Incompatible agents are rejected at check-in with an explicit upgrade message.

### Self-Monitoring

The server monitors its own health and exposes metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `netvantage_db_size_bytes` | Gauge | Database file size |
| `netvantage_db_query_queue_depth` | Gauge | Pending database queries |
| `netvantage_event_bus_queue_depth` | Gauge | Pending async events |
| `netvantage_goroutine_count` | Gauge | Active goroutines |
| `netvantage_disk_free_bytes` | Gauge | Free disk space on data directory |
| `netvantage_uptime_seconds` | Gauge | Server uptime |

Self-monitoring alerts (built-in, always active):
- Disk space < 10% free on data directory
- Database size approaching configured limit
- Event bus queue depth sustained > 1,000
