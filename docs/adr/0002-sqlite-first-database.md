# ADR-0002: SQLite-First Database Strategy

## Status

Accepted

## Date

2025-01-01

## Context

NetVantage targets single-server deployments for home labs and small businesses (Phase 1), scaling to multi-tenant MSP deployments (Phase 2+). The database must:

- Support zero-configuration deployment (no external database server)
- Handle up to 500 devices on a single server (Phase 1 target)
- Provide reliable storage for device inventory, metrics, credentials, and configuration
- Be embeddable in a single Go binary (no CGo if possible)
- Have a migration path to PostgreSQL for larger deployments

## Decision

Use SQLite as the primary database for Phase 1-2, with PostgreSQL + TimescaleDB as an optional upgrade path for Phase 2+.

**SQLite implementation:**
- Library: `modernc.org/sqlite` (pure Go, no CGo dependency)
- WAL mode for concurrent read/write performance
- Per-plugin migration directories for schema isolation
- Thin repository layer (no ORM) with raw SQL
- Reserve `analytics_` table prefix for the Phase 2 Insight plugin

**Migration path:**
- Repository interfaces abstract database operations
- `Store` interface in `internal/store/` enables alternative implementations
- PostgreSQL implementation added in Phase 2 with TimescaleDB hypertables for time-series metrics

## Consequences

### Positive

- Zero external dependencies -- single binary deployment with embedded database
- No CGo requirement -- pure Go build, simplifies cross-compilation
- Excellent performance for single-server workloads (SQLite handles 500+ devices easily)
- WAL mode enables concurrent reads during writes
- Backup is a file copy
- Well-tested, battle-proven database engine

### Negative

- Single-writer limitation (WAL helps but doesn't eliminate write contention)
- No native time-series optimizations (metrics queries will be slower than TimescaleDB)
- Cannot scale horizontally across multiple servers
- Per-plugin migrations add complexity vs a single migration chain

### Neutral

- Repository pattern means all SQL is handwritten (no ORM magic, but more boilerplate)
- PostgreSQL migration requires maintaining two SQL dialects in repository implementations
- `modernc.org/sqlite` is slightly slower than CGo `mattn/go-sqlite3` but avoids build complexity

## Alternatives Considered

### Alternative 1: PostgreSQL Only

Would require users to install and configure PostgreSQL before using NetVantage. Violates the "Time to First Value under 10 minutes" goal and the zero-configuration deployment principle.

### Alternative 2: Embedded Key-Value Store (BoltDB/BadgerDB)

Simpler embedding but loses SQL query capabilities. Would require building a custom query layer for device inventory, metrics aggregation, and credential lookups. Significantly more development effort.

### Alternative 3: CGo SQLite (mattn/go-sqlite3)

Marginally faster but requires CGo toolchain for cross-compilation. On Windows, this means MinGW or MSYS2. On CI, this means separate build environments per platform. The performance difference is negligible for our workload.
