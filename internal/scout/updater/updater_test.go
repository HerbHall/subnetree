package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestFetchExpectedChecksum(t *testing.T) {
	content := []byte("hello world binary content")
	hash := sha256.Sum256(content)
	hashHex := hex.EncodeToString(hash[:])

	checksumFile := fmt.Sprintf("%s  scout_linux_amd64\n%s  scout_windows_amd64.exe\n",
		hashHex, "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(checksumFile))
	}))
	defer srv.Close()

	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		got, err := fetchExpectedChecksum(ctx, srv.URL+"/checksums.txt", "scout_linux_amd64")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != hashHex {
			t.Errorf("got %q, want %q", got, hashHex)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := fetchExpectedChecksum(ctx, srv.URL+"/checksums.txt", "scout_freebsd_amd64")
		if err == nil {
			t.Fatal("expected error for missing binary")
		}
	})
}

func TestApply_VerifiesChecksum(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho updated scout")
	hash := sha256.Sum256(binaryContent)
	hashHex := hex.EncodeToString(hash[:])
	bName := binaryName()

	checksumFile := fmt.Sprintf("%s  %s\n", hashHex, bName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/binary":
			_, _ = w.Write(binaryContent)
		case "/checksums.txt":
			_, _ = w.Write([]byte(checksumFile))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Create a fake "current" binary.
	dir := t.TempDir()
	currentBinary := filepath.Join(dir, "scout")
	if err := os.WriteFile(currentBinary, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	u := &Updater{
		logger:     zap.NewNop(),
		execPath:   currentBinary,
		backupPath: currentBinary + ".bak",
	}

	ctx := context.Background()
	err := u.Apply(ctx, srv.URL+"/binary", srv.URL+"/checksums.txt")
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify the binary was replaced.
	newContent, _ := os.ReadFile(currentBinary)
	if !bytes.Equal(newContent, binaryContent) {
		t.Errorf("binary content = %q, want %q", newContent, binaryContent)
	}

	// Verify backup exists.
	bakContent, _ := os.ReadFile(currentBinary + ".bak")
	if string(bakContent) != "old binary" {
		t.Errorf("backup content = %q, want %q", bakContent, "old binary")
	}
}

func TestApply_ChecksumMismatch(t *testing.T) {
	binaryContent := []byte("some binary")
	bName := binaryName()

	// Provide wrong checksum.
	checksumFile := fmt.Sprintf("0000000000000000000000000000000000000000000000000000000000000000  %s\n", bName)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/binary":
			_, _ = w.Write(binaryContent)
		case "/checksums.txt":
			_, _ = w.Write([]byte(checksumFile))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	currentBinary := filepath.Join(dir, "scout")
	if err := os.WriteFile(currentBinary, []byte("original"), 0o755); err != nil {
		t.Fatal(err)
	}

	u := &Updater{
		logger:     zap.NewNop(),
		execPath:   currentBinary,
		backupPath: currentBinary + ".bak",
	}

	err := u.Apply(context.Background(), srv.URL+"/binary", srv.URL+"/checksums.txt")
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}

	// Verify original binary is untouched.
	content, _ := os.ReadFile(currentBinary)
	if string(content) != "original" {
		t.Errorf("binary should be unchanged, got %q", content)
	}
}

func TestRollback(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "scout")
	bak := exe + ".bak"

	if err := os.WriteFile(exe, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bak, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	u := &Updater{
		logger:     zap.NewNop(),
		execPath:   exe,
		backupPath: bak,
	}

	if err := u.Rollback(); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}

	content, _ := os.ReadFile(exe)
	if string(content) != "old" {
		t.Errorf("after rollback, binary = %q, want %q", content, "old")
	}
}

func TestRollback_NoBackup(t *testing.T) {
	u := &Updater{
		logger:     zap.NewNop(),
		execPath:   "/nonexistent/scout",
		backupPath: "/nonexistent/scout.bak",
	}

	if err := u.Rollback(); err == nil {
		t.Fatal("expected error when no backup exists")
	}
}
