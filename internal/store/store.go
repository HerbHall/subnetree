package store

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/HerbHall/subnetree/pkg/plugin"
	"golang.org/x/mod/semver"
	_ "modernc.org/sqlite" // Pure-Go SQLite driver
)

// ErrNewerSchema is returned when the database was created by a newer version
// of SubNetree than the currently running binary.
var ErrNewerSchema = fmt.Errorf("database was created by a newer version of SubNetree")

// Compile-time interface guard.
var _ plugin.Store = (*SQLiteStore)(nil)

// SQLiteStore implements plugin.Store backed by SQLite via modernc.org/sqlite.
type SQLiteStore struct {
	db   *sql.DB
	mu   sync.Mutex // Serialize migrations
	once sync.Once  // Ensure _migrations table created once
}

// New opens (or creates) a SQLite database at the given path and applies
// recommended pragmas for WAL mode, foreign keys, and performance.
// Returns the concrete type; callers assign to plugin.Store where needed.
func New(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}

	// SQLite performs best with a single write connection. WAL enables concurrent readers.
	db.SetMaxOpenConns(1)

	// Verify the connection works.
	if err := db.PingContext(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite %q: %w", path, err)
	}

	// Apply recommended pragmas (modernc.org/sqlite requires SQL statements, not DSN params).
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA cache_size=-20000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("exec %q: %w", p, err)
		}
	}

	return &SQLiteStore{db: db}, nil
}

// DB returns the underlying *sql.DB for direct queries.
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// Tx executes fn within a database transaction. The transaction is
// committed if fn returns nil, rolled back otherwise.
func (s *SQLiteStore) Tx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original: %w)", rbErr, err)
		}
		return err
	}

	return tx.Commit()
}

// Migrate runs pending migrations for the named plugin. Already-applied
// migrations (tracked in the shared _migrations table) are skipped.
// Migrations must be provided in ascending Version order.
func (s *SQLiteStore) Migrate(ctx context.Context, pluginName string, migrations []plugin.Migration) error {
	if err := s.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, m := range migrations {
		applied, err := s.isMigrationApplied(ctx, pluginName, m.Version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := s.applyMigration(ctx, pluginName, m); err != nil {
			return fmt.Errorf("migration %s/%d (%s): %w", pluginName, m.Version, m.Description, err)
		}
	}

	return nil
}

// Close closes the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// CheckVersion compares the running binary version against the version stored
// in the database. It prevents an older binary from opening a database created
// by a newer version, which could corrupt data. The special version "dev"
// always passes (both as stored and as current).
func (s *SQLiteStore) CheckVersion(ctx context.Context, currentVersion string) error {
	if err := s.ensureSchemaMetaTable(ctx); err != nil {
		return fmt.Errorf("ensure schema meta table: %w", err)
	}

	var stored string
	err := s.db.QueryRowContext(ctx,
		"SELECT app_version FROM _schema_meta WHERE id = 1",
	).Scan(&stored)

	if err == sql.ErrNoRows {
		// First run: record the current version.
		_, err = s.db.ExecContext(ctx,
			"INSERT INTO _schema_meta (id, app_version, updated_at) VALUES (1, ?, CURRENT_TIMESTAMP)",
			currentVersion,
		)
		if err != nil {
			return fmt.Errorf("insert schema version: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("query schema version: %w", err)
	}

	// "dev" always passes -- useful for local development.
	if stored == "dev" || currentVersion == "dev" {
		_, err = s.db.ExecContext(ctx,
			"UPDATE _schema_meta SET app_version = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1",
			currentVersion,
		)
		if err != nil {
			return fmt.Errorf("update schema version: %w", err)
		}
		return nil
	}

	cur := normalizeVersion(currentVersion)
	sto := normalizeVersion(stored)

	if semver.Compare(cur, sto) < 0 {
		return fmt.Errorf("%w: database=%s, binary=%s", ErrNewerSchema, stored, currentVersion)
	}

	// Current >= stored: update the stored version.
	if semver.Compare(cur, sto) > 0 {
		_, err = s.db.ExecContext(ctx,
			"UPDATE _schema_meta SET app_version = ?, updated_at = CURRENT_TIMESTAMP WHERE id = 1",
			currentVersion,
		)
		if err != nil {
			return fmt.Errorf("update schema version: %w", err)
		}
	}

	return nil
}

// ensureSchemaMetaTable creates the _schema_meta table if it doesn't exist.
func (s *SQLiteStore) ensureSchemaMetaTable(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS _schema_meta (
			id           INTEGER  PRIMARY KEY CHECK (id = 1),
			app_version  TEXT     NOT NULL,
			updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// normalizeVersion ensures the version string has a "v" prefix for semver comparison.
func normalizeVersion(v string) string {
	if v != "" && v[0] != 'v' {
		return "v" + v
	}
	return v
}

// ensureMigrationsTable creates the shared _migrations tracking table if it
// doesn't already exist. Safe to call multiple times (uses sync.Once).
func (s *SQLiteStore) ensureMigrationsTable(ctx context.Context) error {
	var err error
	s.once.Do(func() {
		_, err = s.db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS _migrations (
				plugin_name TEXT    NOT NULL,
				version     INTEGER NOT NULL,
				description TEXT    NOT NULL,
				applied_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (plugin_name, version)
			)
		`)
	})
	return err
}

func (s *SQLiteStore) isMigrationApplied(ctx context.Context, pluginName string, version int) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM _migrations WHERE plugin_name = ? AND version = ?",
		pluginName, version,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check migration %s/%d: %w", pluginName, version, err)
	}
	return count > 0, nil
}

func (s *SQLiteStore) applyMigration(ctx context.Context, pluginName string, m plugin.Migration) error {
	return s.Tx(ctx, func(tx *sql.Tx) error {
		if err := m.Up(tx); err != nil {
			return err
		}

		_, err := tx.ExecContext(ctx,
			"INSERT INTO _migrations (plugin_name, version, description) VALUES (?, ?, ?)",
			pluginName, m.Version, m.Description,
		)
		return err
	})
}
