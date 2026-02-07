package backup_test

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/HerbHall/subnetree/internal/backup"
	_ "modernc.org/sqlite"
)

// createTestDB creates a SQLite database with a test table and returns the path.
func createTestDB(t *testing.T, dir string) string {
	t.Helper()

	dbPath := filepath.Join(dir, "subnetree.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE test_data (id INTEGER PRIMARY KEY, name TEXT);
		INSERT INTO test_data (id, name) VALUES (1, 'alice'), (2, 'bob');
	`)
	if err != nil {
		t.Fatal(err)
	}

	return dbPath
}

// createTestConfig writes a small config file and returns the path.
func createTestConfig(t *testing.T, dir string) string {
	t.Helper()

	cfgPath := filepath.Join(dir, "subnetree.yaml")
	if err := os.WriteFile(cfgPath, []byte("server:\n  port: 8080\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	return cfgPath
}

// verifyDBContents checks that the restored database has expected data.
func verifyDBContents(t *testing.T, dbPath string) {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM test_data").Scan(&count); err != nil {
		t.Fatalf("querying restored DB: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows, got %d", count)
	}

	var name string
	if err := db.QueryRow("SELECT name FROM test_data WHERE id = 1").Scan(&name); err != nil {
		t.Fatalf("querying row: %v", err)
	}
	if name != "alice" {
		t.Fatalf("expected name 'alice', got %q", name)
	}
}

func TestBackupRestore(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) (dbPath, configPath, archivePath, restoreDir string)
		backupErr  string
		restoreErr string
		force      bool
		verify     func(t *testing.T, restoreDir string)
	}{
		{
			name: "round trip with config",
			setup: func(t *testing.T) (string, string, string, string) {
				srcDir := t.TempDir()
				archiveDir := t.TempDir()
				restoreDir := t.TempDir()
				dbPath := createTestDB(t, srcDir)
				cfgPath := createTestConfig(t, srcDir)
				return dbPath, cfgPath, filepath.Join(archiveDir, "backup.tar.gz"), restoreDir
			},
			verify: func(t *testing.T, restoreDir string) {
				verifyDBContents(t, filepath.Join(restoreDir, "subnetree.db"))
				// Config should also be restored.
				data, err := os.ReadFile(filepath.Join(restoreDir, "subnetree.yaml"))
				if err != nil {
					t.Fatalf("config not restored: %v", err)
				}
				if len(data) == 0 {
					t.Fatal("restored config is empty")
				}
			},
		},
		{
			name: "round trip without config",
			setup: func(t *testing.T) (string, string, string, string) {
				srcDir := t.TempDir()
				archiveDir := t.TempDir()
				restoreDir := t.TempDir()
				dbPath := createTestDB(t, srcDir)
				return dbPath, "", filepath.Join(archiveDir, "backup.tar.gz"), restoreDir
			},
			verify: func(t *testing.T, restoreDir string) {
				verifyDBContents(t, filepath.Join(restoreDir, "subnetree.db"))
			},
		},
		{
			name: "missing database",
			setup: func(t *testing.T) (string, string, string, string) {
				archiveDir := t.TempDir()
				restoreDir := t.TempDir()
				return filepath.Join(t.TempDir(), "nonexistent.db"), "", filepath.Join(archiveDir, "backup.tar.gz"), restoreDir
			},
			backupErr: "database file not found",
		},
		{
			name: "no force existing DB",
			setup: func(t *testing.T) (string, string, string, string) {
				srcDir := t.TempDir()
				archiveDir := t.TempDir()
				restoreDir := t.TempDir()
				dbPath := createTestDB(t, srcDir)
				// Pre-create a file in the restore dir to trigger conflict.
				createTestDB(t, restoreDir)
				return dbPath, "", filepath.Join(archiveDir, "backup.tar.gz"), restoreDir
			},
			restoreErr: "file already exists",
		},
		{
			name:  "force existing DB",
			force: true,
			setup: func(t *testing.T) (string, string, string, string) {
				srcDir := t.TempDir()
				archiveDir := t.TempDir()
				restoreDir := t.TempDir()
				dbPath := createTestDB(t, srcDir)
				// Pre-create a file in the restore dir.
				createTestDB(t, restoreDir)
				return dbPath, "", filepath.Join(archiveDir, "backup.tar.gz"), restoreDir
			},
			verify: func(t *testing.T, restoreDir string) {
				verifyDBContents(t, filepath.Join(restoreDir, "subnetree.db"))
			},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dbPath, cfgPath, archivePath, restoreDir := tc.setup(t)

			err := backup.Backup(ctx, dbPath, cfgPath, archivePath)
			if tc.backupErr != "" {
				if err == nil {
					t.Fatalf("expected backup error containing %q, got nil", tc.backupErr)
				}
				if !contains(err.Error(), tc.backupErr) {
					t.Fatalf("expected backup error containing %q, got %q", tc.backupErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected backup error: %v", err)
			}

			err = backup.Restore(ctx, archivePath, restoreDir, tc.force)
			if tc.restoreErr != "" {
				if err == nil {
					t.Fatalf("expected restore error containing %q, got nil", tc.restoreErr)
				}
				if !contains(err.Error(), tc.restoreErr) {
					t.Fatalf("expected restore error containing %q, got %q", tc.restoreErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected restore error: %v", err)
			}

			if tc.verify != nil {
				tc.verify(t, restoreDir)
			}
		})
	}
}

func TestRestore_CorruptArchive(t *testing.T) {
	dir := t.TempDir()
	corruptPath := filepath.Join(dir, "corrupt.tar.gz")
	if err := os.WriteFile(corruptPath, []byte("not a valid gzip"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := backup.Restore(context.Background(), corruptPath, t.TempDir(), false)
	if err == nil {
		t.Fatal("expected error for corrupt archive, got nil")
	}
}

func TestRestore_PathTraversal(t *testing.T) {
	// Create an archive with a path traversal entry.
	archiveDir := t.TempDir()
	archivePath := filepath.Join(archiveDir, "evil.tar.gz")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Write a malicious entry that tries to escape.
	hdr := &tar.Header{
		Name: "../../../etc/evil.db",
		Size: 4,
		Mode: 0o644,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("evil")); err != nil {
		t.Fatal(err)
	}

	tw.Close()
	gw.Close()
	f.Close()

	err = backup.Restore(context.Background(), archivePath, t.TempDir(), false)
	if err == nil {
		t.Fatal("expected path traversal error, got nil")
	}
	if !contains(err.Error(), "path traversal") {
		t.Fatalf("expected path traversal error, got %q", err.Error())
	}
}

func TestRestore_NoDBInArchive(t *testing.T) {
	// Create an archive with only a non-.db file.
	archiveDir := t.TempDir()
	archivePath := filepath.Join(archiveDir, "nodb.tar.gz")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name: "config.yaml",
		Size: 5,
		Mode: 0o644,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}

	tw.Close()
	gw.Close()
	f.Close()

	err = backup.Restore(context.Background(), archivePath, t.TempDir(), false)
	if err == nil {
		t.Fatal("expected error for archive without .db file, got nil")
	}
	if !contains(err.Error(), "does not contain a .db file") {
		t.Fatalf("expected .db file error, got %q", err.Error())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
