//go:build windows

package updater

import "os"

// atomicReplace on Windows handles the fact that a running .exe cannot be renamed.
// Strategy: rename the running binary to .bak (Windows allows renaming a locked file
// but not deleting it), then move the new binary into place.
func atomicReplace(currentPath, backupPath, newPath string) error {
	// Remove any stale backup.
	os.Remove(backupPath)

	// Rename the running binary to .bak.
	// Windows allows renaming a running executable but not overwriting it.
	if err := os.Rename(currentPath, backupPath); err != nil {
		return err
	}

	// Move new binary into the original path.
	if err := os.Rename(newPath, currentPath); err != nil {
		// Attempt to restore on failure.
		_ = os.Rename(backupPath, currentPath)
		return err
	}
	return nil
}
