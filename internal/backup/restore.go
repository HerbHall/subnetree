package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Restore extracts a backup archive to the target directory.
// It refuses to overwrite existing files unless force is true.
func Restore(_ context.Context, archivePath, targetDir string, force bool) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("decompressing archive: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	// Ensure target directory exists.
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("creating target directory: %w", err)
	}

	foundDB := false

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading archive entry: %w", err)
		}

		// Security: reject entries that escape the target directory.
		if err := validateTarEntry(hdr.Name, targetDir); err != nil {
			return err
		}

		if strings.HasSuffix(hdr.Name, ".db") {
			foundDB = true
		}

		destPath := filepath.Join(targetDir, filepath.Clean(hdr.Name)) //nolint:gosec // G305: path traversal checked by validateTarEntry above

		// Check for existing files when force is disabled.
		if !force {
			if _, err := os.Stat(destPath); err == nil {
				return fmt.Errorf("file already exists (use --force to overwrite): %s", destPath)
			}
		}

		if err := extractFile(tr, destPath, hdr); err != nil {
			return fmt.Errorf("extracting %s: %w", hdr.Name, err)
		}
	}

	if !foundDB {
		return fmt.Errorf("invalid backup: archive does not contain a .db file")
	}

	return nil
}

// validateTarEntry checks that a tar entry name does not escape the target
// directory via path traversal.
func validateTarEntry(name, targetDir string) error {
	// Reject absolute paths.
	if filepath.IsAbs(name) {
		return fmt.Errorf("path traversal detected: absolute path %q", name)
	}

	// Clean the path and check for directory escape.
	cleaned := filepath.Clean(name)
	if strings.HasPrefix(cleaned, "..") {
		return fmt.Errorf("path traversal detected: %q", name)
	}

	// Double-check: resolved path must be within target.
	dest := filepath.Join(targetDir, cleaned)
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("resolving target directory: %w", err)
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("resolving destination path: %w", err)
	}
	if !strings.HasPrefix(absDest, absTarget+string(filepath.Separator)) && absDest != absTarget {
		return fmt.Errorf("path traversal detected: %q resolves outside target", name)
	}

	return nil
}

// extractFile writes a single tar entry to disk.
func extractFile(tr *tar.Reader, destPath string, hdr *tar.Header) error {
	switch hdr.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(destPath, os.FileMode(hdr.Mode&0o777)) //nolint:gosec // G115: mode bits safely within uint32 range
	case tar.TypeReg:
		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode&0o777)) //nolint:gosec // G115: mode bits safely within uint32 range
		if err != nil {
			return err
		}
		defer out.Close()

		// Limit copy size to prevent decompression bombs.
		const maxFileSize = 10 << 30 // 10 GiB
		_, err = io.Copy(out, io.LimitReader(tr, maxFileSize))
		return err
	default:
		// Skip unsupported entry types (symlinks, etc.).
		return nil
	}
}
