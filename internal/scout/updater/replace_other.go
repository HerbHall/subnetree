//go:build !windows

package updater

import "os"

// atomicReplace backs up the current binary and moves the new one into place.
// On Unix, os.Rename is atomic when src and dst are on the same filesystem.
func atomicReplace(currentPath, backupPath, newPath string) error {
	// Remove any stale backup.
	os.Remove(backupPath)

	// Back up current binary.
	if err := os.Rename(currentPath, backupPath); err != nil {
		return err
	}

	// Move new binary into place.
	if err := os.Rename(newPath, currentPath); err != nil {
		// Attempt to restore backup on failure.
		_ = os.Rename(backupPath, currentPath)
		return err
	}
	return nil
}
