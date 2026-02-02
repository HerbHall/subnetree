## Database Layer

### Architecture

Shared connection pool with per-plugin schema ownership. Each plugin owns its own tables (prefixed with plugin name) but shares a single database connection.

### Store Interface

```go
type Store interface {
    DB() *sql.DB
    Tx(ctx context.Context, fn func(tx *sql.Tx) error) error
    Migrate(ctx context.Context, pluginName string, migrations []Migration) error
}

type Migration struct {
    Version     int
    Description string
    Up          func(tx *sql.Tx) error
}
```

### SQLite Configuration (Phase 1)

Driver: `modernc.org/sqlite` (pure Go, no CGo dependency)

Connection pragmas:
- `_journal_mode=WAL` -- Concurrent reads during writes
- `_busy_timeout=5000` -- Wait up to 5s for locks instead of failing immediately
- `_synchronous=NORMAL` -- Safe with WAL mode, better write performance
- `_foreign_keys=ON` -- Enforce referential integrity
- `_cache_size=-20000` -- 20MB page cache

`MaxOpenConns(1)` -- SQLite performs best with a single write connection. WAL enables concurrent readers.

### Migration Tracking

A shared `_migrations` table tracks applied migrations per plugin:

```sql
CREATE TABLE _migrations (
    plugin_name TEXT NOT NULL,
    version     INTEGER NOT NULL,
    description TEXT NOT NULL,
    applied_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (plugin_name, version)
);
```

### Repository Pattern

- **Shared interfaces** in `internal/services/` -- `DeviceRepository`, `CredentialProvider`, `AgentManager`
- **Private implementations** in each plugin package -- SQLite-specific query code
- **No ORM** -- Raw SQL with thin repository layer. Queries are straightforward CRUD, and raw SQL provides performance transparency and debugging clarity.

### PostgreSQL Migration Path (Phase 2+)

- Repository interfaces remain the same; only implementations change
- TimescaleDB hypertables for time-series metrics (Pulse module)
- Continuous aggregates for dashboard rollup queries
- Retention policies for automatic data lifecycle
- Connection pooling via pgxpool
