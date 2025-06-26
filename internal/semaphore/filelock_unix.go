//go:build !windows

package semaphore

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// fileLock provides file locking functionality using Unix flock
type fileLock struct {
	file *os.File
}

// newFileLock creates a new file lock
func newFileLock(path string) (*fileLock, error) {
	lockPath := path + ".lock"

	// Ensure directory exists
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open or create the lock file
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}
	return &fileLock{file: file}, nil
}

// lock acquires an exclusive lock on the file
func (fl *fileLock) lock() error {
	if err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	return nil
}

// unlock releases the lock on the file
func (fl *fileLock) unlock() error {
	if err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}

// close closes the file
func (fl *fileLock) close() error {
	return fl.file.Close()
}
