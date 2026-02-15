package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/HerbHall/subnetree/pkg/plugin"
)

func tempDB(t *testing.T) *SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	s, err := New(path)
	if err != nil {
		t.Fatalf("New(%q): %v", path, err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestNew_creates_database(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.db")
	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func TestNew_invalid_path(t *testing.T) {
	_, err := New("/nonexistent/path/to/db")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestDB_returns_connection(t *testing.T) {
	s := tempDB(t)
	if s.DB() == nil {
		t.Error("DB() returned nil")
	}
}

func TestTx_commit(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	_, err := s.DB().ExecContext(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	err = s.Tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO test (id, name) VALUES (1, 'alice')")
		return err
	})
	if err != nil {
		t.Fatalf("Tx commit: %v", err)
	}

	var name string
	err = s.DB().QueryRowContext(ctx, "SELECT name FROM test WHERE id = 1").Scan(&name)
	if err != nil {
		t.Fatalf("query after commit: %v", err)
	}
	if name != "alice" {
		t.Errorf("got name %q, want %q", name, "alice")
	}
}

func TestTx_rollback(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	_, err := s.DB().ExecContext(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	err = s.Tx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO test (id, name) VALUES (1, 'bob')")
		if err != nil {
			return err
		}
		return sql.ErrNoRows // Simulate an error to trigger rollback
	})
	if err != sql.ErrNoRows {
		t.Fatalf("expected ErrNoRows, got %v", err)
	}

	var count int
	err = s.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM test").Scan(&count)
	if err != nil {
		t.Fatalf("count after rollback: %v", err)
	}
	if count != 0 {
		t.Errorf("got count %d after rollback, want 0", count)
	}
}

func TestMigrate_applies_in_order(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	migrations := []plugin.Migration{
		{
			Version:     1,
			Description: "create devices table",
			Up: func(tx *sql.Tx) error {
				_, err := tx.Exec("CREATE TABLE recon_devices (id INTEGER PRIMARY KEY, name TEXT)")
				return err
			},
		},
		{
			Version:     2,
			Description: "add ip column",
			Up: func(tx *sql.Tx) error {
				_, err := tx.Exec("ALTER TABLE recon_devices ADD COLUMN ip TEXT")
				return err
			},
		},
	}

	if err := s.Migrate(ctx, "recon", migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Verify table and columns exist.
	_, err := s.DB().ExecContext(ctx, "INSERT INTO recon_devices (id, name, ip) VALUES (1, 'switch1', '10.0.0.1')")
	if err != nil {
		t.Fatalf("insert after migration: %v", err)
	}

	// Verify migration tracking.
	var count int
	err = s.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM _migrations WHERE plugin_name = 'recon'").Scan(&count)
	if err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count != 2 {
		t.Errorf("got %d migration records, want 2", count)
	}
}

func TestMigrate_skips_applied(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	callCount := 0
	migrations := []plugin.Migration{
		{
			Version:     1,
			Description: "create table",
			Up: func(tx *sql.Tx) error {
				callCount++
				_, err := tx.Exec("CREATE TABLE test_skip (id INTEGER)")
				return err
			},
		},
	}

	// Apply once.
	if err := s.Migrate(ctx, "test", migrations); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}

	// Apply again -- should be a no-op.
	if err := s.Migrate(ctx, "test", migrations); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if callCount != 1 {
		t.Errorf("migration ran again: callCount=%d, want 1", callCount)
	}
}

func TestMigrate_different_plugins_isolated(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	reconMigrations := []plugin.Migration{
		{Version: 1, Description: "recon table", Up: func(tx *sql.Tx) error {
			_, err := tx.Exec("CREATE TABLE recon_data (id INTEGER)")
			return err
		}},
	}
	pulseMigrations := []plugin.Migration{
		{Version: 1, Description: "pulse table", Up: func(tx *sql.Tx) error {
			_, err := tx.Exec("CREATE TABLE pulse_data (id INTEGER)")
			return err
		}},
	}

	if err := s.Migrate(ctx, "recon", reconMigrations); err != nil {
		t.Fatalf("recon Migrate: %v", err)
	}
	if err := s.Migrate(ctx, "pulse", pulseMigrations); err != nil {
		t.Fatalf("pulse Migrate: %v", err)
	}

	// Both tables should exist.
	for _, table := range []string{"recon_data", "pulse_data"} {
		var name string
		err := s.DB().QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestMigrate_failure_rolls_back(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	migrations := []plugin.Migration{
		{Version: 1, Description: "will fail", Up: func(tx *sql.Tx) error {
			_, err := tx.Exec("INVALID SQL STATEMENT")
			return err
		}},
	}

	err := s.Migrate(ctx, "bad", migrations)
	if err == nil {
		t.Fatal("expected error from bad migration, got nil")
	}

	// Verify no migration was recorded.
	var count int
	err = s.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM _migrations WHERE plugin_name = 'bad'").Scan(&count)
	if err != nil {
		t.Fatalf("count after failed migration: %v", err)
	}
	if count != 0 {
		t.Errorf("migration was recorded despite failure: count=%d", count)
	}
}

func TestMigrate_partial_failure_preserves_earlier(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	migrations := []plugin.Migration{
		{Version: 1, Description: "ok migration", Up: func(tx *sql.Tx) error {
			_, err := tx.Exec("CREATE TABLE partial_test (id INTEGER)")
			return err
		}},
		{Version: 2, Description: "bad migration", Up: func(tx *sql.Tx) error {
			_, err := tx.Exec("INVALID SQL")
			return err
		}},
	}

	err := s.Migrate(ctx, "partial", migrations)
	if err == nil {
		t.Fatal("expected error from partial migration")
	}

	// First migration should be committed.
	var count int
	err = s.DB().QueryRowContext(ctx, "SELECT COUNT(*) FROM _migrations WHERE plugin_name = 'partial'").Scan(&count)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 committed migration, got %d", count)
	}
}

func TestClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "close.db")
	s, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// After close, queries should fail.
	err = s.DB().PingContext(context.Background())
	if err == nil {
		t.Error("expected error after Close, got nil")
	}
}

func TestWAL_mode_enabled(t *testing.T) {
	s := tempDB(t)
	var mode string
	err := s.DB().QueryRowContext(context.Background(), "PRAGMA journal_mode").Scan(&mode)
	if err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want %q", mode, "wal")
	}
}

func TestForeignKeys_enabled(t *testing.T) {
	s := tempDB(t)
	var fk int
	err := s.DB().QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&fk)
	if err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestCheckVersion_FirstRun(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	if err := s.CheckVersion(ctx, "0.4.0"); err != nil {
		t.Fatalf("CheckVersion first run: %v", err)
	}

	// Verify version was stored.
	var stored string
	err := s.DB().QueryRowContext(ctx, "SELECT app_version FROM _schema_meta WHERE id = 1").Scan(&stored)
	if err != nil {
		t.Fatalf("query stored version: %v", err)
	}
	if stored != "0.4.0" {
		t.Errorf("stored version = %q, want %q", stored, "0.4.0")
	}
}

func TestCheckVersion_SameVersion(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	if err := s.CheckVersion(ctx, "0.4.0"); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := s.CheckVersion(ctx, "0.4.0"); err != nil {
		t.Fatalf("second call with same version: %v", err)
	}
}

func TestCheckVersion_NewerBinary(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	if err := s.CheckVersion(ctx, "0.4.0"); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := s.CheckVersion(ctx, "0.5.0"); err != nil {
		t.Fatalf("upgrade to 0.5.0: %v", err)
	}

	// Verify stored version was updated.
	var stored string
	err := s.DB().QueryRowContext(ctx, "SELECT app_version FROM _schema_meta WHERE id = 1").Scan(&stored)
	if err != nil {
		t.Fatalf("query stored version: %v", err)
	}
	if stored != "0.5.0" {
		t.Errorf("stored version = %q, want %q", stored, "0.5.0")
	}
}

func TestCheckVersion_OlderBinary_Rejected(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	if err := s.CheckVersion(ctx, "0.5.0"); err != nil {
		t.Fatalf("first call: %v", err)
	}

	err := s.CheckVersion(ctx, "0.4.0")
	if err == nil {
		t.Fatal("expected error when running older binary against newer database")
	}
	if !errors.Is(err, ErrNewerSchema) {
		t.Errorf("expected ErrNewerSchema, got: %v", err)
	}
}

func TestCheckVersion_DevAlwaysPasses(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	// dev -> 0.5.0 -> dev: all should pass.
	if err := s.CheckVersion(ctx, "dev"); err != nil {
		t.Fatalf("dev first run: %v", err)
	}
	if err := s.CheckVersion(ctx, "0.5.0"); err != nil {
		t.Fatalf("dev -> 0.5.0: %v", err)
	}
	if err := s.CheckVersion(ctx, "dev"); err != nil {
		t.Fatalf("0.5.0 -> dev: %v", err)
	}
}

func TestCheckVersion_PatchUpgrade(t *testing.T) {
	s := tempDB(t)
	ctx := context.Background()

	if err := s.CheckVersion(ctx, "0.4.0"); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := s.CheckVersion(ctx, "0.4.1"); err != nil {
		t.Fatalf("patch upgrade 0.4.0 -> 0.4.1: %v", err)
	}
}
