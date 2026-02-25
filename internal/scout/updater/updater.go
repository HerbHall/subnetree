package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.uber.org/zap"
)

// Updater handles binary self-update for Scout.
type Updater struct {
	logger     *zap.Logger
	execPath   string
	backupPath string
}

// New creates an Updater. Resolves the current executable path.
func New(logger *zap.Logger) (*Updater, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return nil, fmt.Errorf("resolve symlinks: %w", err)
	}
	return &Updater{
		logger:     logger,
		execPath:   exe,
		backupPath: exe + ".bak",
	}, nil
}

// Apply downloads binary from downloadURL, verifies SHA256 against checksumURL,
// and atomically replaces the current binary. The old binary is saved as .bak.
func (u *Updater) Apply(ctx context.Context, downloadURL, checksumURL string) error {
	u.logger.Info("downloading update",
		zap.String("url", downloadURL),
	)

	// Download to a temp file in the same directory (required for atomic rename).
	dir := filepath.Dir(u.execPath)
	tmpFile, err := os.CreateTemp(dir, "scout-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath) // clean up on failure; no-op if renamed
	}()

	// Download the binary.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("create download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Write to temp file and compute SHA256 simultaneously.
	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmpFile, hasher), resp.Body)
	if err != nil {
		return fmt.Errorf("write update binary: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	u.logger.Info("download complete", zap.Int64("bytes", written))

	// Verify checksum.
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	expectedHash, err := fetchExpectedChecksum(ctx, checksumURL, binaryName())
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	u.logger.Info("checksum verified", zap.String("sha256", actualHash))

	// Make the downloaded file executable (Unix).
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		u.logger.Warn("failed to chmod update binary", zap.Error(err))
	}

	// Atomic replace: backup current, move new into place.
	if err := atomicReplace(u.execPath, u.backupPath, tmpPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}

	u.logger.Info("binary replaced successfully",
		zap.String("path", u.execPath),
		zap.String("backup", u.backupPath),
	)
	return nil
}

// Rollback restores the previous binary from the .bak file.
func (u *Updater) Rollback() error {
	if _, err := os.Stat(u.backupPath); os.IsNotExist(err) {
		return fmt.Errorf("no backup found at %s", u.backupPath)
	}
	return os.Rename(u.backupPath, u.execPath)
}

// binaryName returns the expected binary filename for the current platform.
func binaryName() string {
	name := fmt.Sprintf("scout_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// fetchExpectedChecksum fetches checksums.txt from the given URL and returns
// the SHA256 hash for the specified binary name.
func fetchExpectedChecksum(ctx context.Context, checksumURL, binary string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("create checksum request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read checksums: %w", err)
	}

	// GoReleaser checksums.txt format: "<sha256>  <filename>"
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == binary {
			return parts[0], nil
		}
	}

	return "", fmt.Errorf("binary %q not found in checksums", binary)
}
