//go:build windows

package filemanager

import (
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

// createLock creates a file lock for the given path.
// On Windows, we use a separate lock file to avoid conflicts with rename operations.
func createLock(path string) *flock.Flock {
	lockPath := getLockPath(path)

	// Ensure the directory exists for the lock file
	dir := filepath.Dir(lockPath)
	_ = os.MkdirAll(dir, 0o755)

	return flock.New(lockPath)
}

// cleanupLockFile removes the lock file if it exists and is old enough.
// On Windows, we use separate lock files that need to be cleaned up.
func cleanupLockFile(path string) {
	lockPath := getLockPath(path)

	// Try to remove the lock file
	// We don't care if it fails (it might be in use by another process)
	info, err := os.Stat(lockPath)
	if err == nil && time.Since(info.ModTime()) > 5*time.Second {
		// Only remove old lock files to avoid race conditions
		_ = os.Remove(lockPath)
	}
}
