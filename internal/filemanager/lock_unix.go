//go:build !windows

package filemanager

import "github.com/gofrs/flock"

// createLock creates a file lock for the given path.
// On Unix systems, we can lock the actual file.
func createLock(path string) *flock.Flock {
	return flock.New(path)
}

// cleanupLockFile is a no-op on Unix systems since we lock the actual file.
func cleanupLockFile(path string) {
	// No-op on Unix
}
