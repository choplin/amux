// Package filemanager provides thread-safe and process-safe file operations with CAS support.
package filemanager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
	"gopkg.in/yaml.v3"
)

// ErrConcurrentModification is returned when a file has been modified since it was read
var ErrConcurrentModification = errors.New("file was modified concurrently")

// ErrLockTimeout is returned when acquiring a file lock times out
var ErrLockTimeout = errors.New("timeout acquiring file lock")

// FileInfo represents metadata about a file used for CAS operations
type FileInfo struct {
	Path    string
	ModTime time.Time
	Size    int64
}

// UpdateFunc is a function that modifies data in-place
type UpdateFunc[T any] func(data *T) error

// Manager provides thread-safe and process-safe file operations with CAS support
type Manager[T any] struct {
	// lockTimeout is the maximum time to wait for a file lock
	lockTimeout time.Duration
}

// NewManager creates a new file manager with default settings
func NewManager[T any]() *Manager[T] {
	return &Manager[T]{
		lockTimeout: 5 * time.Second,
	}
}

// NewManagerWithTimeout creates a new file manager with custom lock timeout
func NewManagerWithTimeout[T any](timeout time.Duration) *Manager[T] {
	return &Manager[T]{
		lockTimeout: timeout,
	}
}

// Read reads a file with a shared lock
func (m *Manager[T]) Read(ctx context.Context, path string) (*T, *FileInfo, error) {
	// Check if file exists first
	if _, err := os.Stat(path); err != nil {
		return nil, nil, err
	}

	// Create a file lock
	lock := flock.New(path)

	// Try to acquire shared lock with timeout
	lockCtx, cancel := context.WithTimeout(ctx, m.lockTimeout)
	defer cancel()

	locked, err := lock.TryRLockContext(lockCtx, 100*time.Millisecond)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire read lock: %w", err)
	}
	if !locked {
		return nil, nil, ErrLockTimeout
	}
	defer func() { _ = lock.Unlock() }()

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	// Get file info for CAS
	stat, err := os.Stat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat file: %w", err)
	}

	info := &FileInfo{
		Path:    path,
		ModTime: stat.ModTime(),
		Size:    stat.Size(),
	}

	// Unmarshal the data
	var result T
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	return &result, info, nil
}

// Write writes a file with an exclusive lock (no CAS check)
func (m *Manager[T]) Write(ctx context.Context, path string, data *T) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create a file lock
	lock := flock.New(path)

	// Try to acquire exclusive lock with timeout
	lockCtx, cancel := context.WithTimeout(ctx, m.lockTimeout)
	defer cancel()

	locked, err := lock.TryLockContext(lockCtx, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to acquire write lock: %w", err)
	}
	if !locked {
		return ErrLockTimeout
	}
	defer func() { _ = lock.Unlock() }()

	// Marshal the data
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml: %w", err)
	}

	// Write atomically using temp file + rename
	// Use a unique temp file name to avoid conflicts on Windows
	tempFile := fmt.Sprintf("%s.%d.%d.tmp", path, os.Getpid(), time.Now().UnixNano())
	if err := os.WriteFile(tempFile, yamlData, 0o644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Sync to ensure data is written to disk
	// We need to open for write to be able to sync
	if f, err := os.OpenFile(tempFile, os.O_RDWR, 0o644); err == nil {
		_ = f.Sync()
		_ = f.Close()
	}

	// Atomic rename
	if err := atomicRename(tempFile, path); err != nil {
		_ = os.Remove(tempFile) // Clean up on failure
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// WriteWithCAS writes a file only if it hasn't changed since the provided FileInfo
func (m *Manager[T]) WriteWithCAS(ctx context.Context, path string, data *T, expectedInfo *FileInfo) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create a file lock
	lock := flock.New(path)

	// Try to acquire exclusive lock with timeout
	lockCtx, cancel := context.WithTimeout(ctx, m.lockTimeout)
	defer cancel()

	locked, err := lock.TryLockContext(lockCtx, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to acquire write lock: %w", err)
	}
	if !locked {
		return ErrLockTimeout
	}
	defer func() { _ = lock.Unlock() }()

	// Check if file has been modified
	if expectedInfo != nil {
		stat, err := os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to stat file: %w", err)
		}

		// If file exists, check modification time and size
		if err == nil {
			if !stat.ModTime().Equal(expectedInfo.ModTime) || stat.Size() != expectedInfo.Size {
				return ErrConcurrentModification
			}
		}
	}

	// Marshal the data
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml: %w", err)
	}

	// Write atomically using temp file + rename
	// Use a unique temp file name to avoid conflicts on Windows
	tempFile := fmt.Sprintf("%s.%d.%d.tmp", path, os.Getpid(), time.Now().UnixNano())
	if err := os.WriteFile(tempFile, yamlData, 0o644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Sync to ensure data is written to disk
	// We need to open for write to be able to sync
	if f, err := os.OpenFile(tempFile, os.O_RDWR, 0o644); err == nil {
		_ = f.Sync()
		_ = f.Close()
	}

	// Atomic rename
	if err := atomicRename(tempFile, path); err != nil {
		_ = os.Remove(tempFile) // Clean up on failure
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// Update reads a file, applies an update function, and writes it back with CAS
func (m *Manager[T]) Update(ctx context.Context, path string, updateFunc UpdateFunc[T]) error {
	const maxRetries = 10

	for i := 0; i < maxRetries; i++ {
		// Read current data
		data, info, err := m.Read(ctx, path)
		if err != nil {
			if os.IsNotExist(err) {
				// File doesn't exist, create new
				var newData T
				if err := updateFunc(&newData); err != nil {
					return fmt.Errorf("update function failed: %w", err)
				}
				if err := m.Write(ctx, path, &newData); err != nil {
					if errors.Is(err, ErrConcurrentModification) {
						continue // Retry
					}
					return err
				}
				return nil
			}
			return fmt.Errorf("failed to read file: %w", err)
		}

		// Apply update
		if err := updateFunc(data); err != nil {
			return fmt.Errorf("update function failed: %w", err)
		}

		// Try to write with CAS
		if err := m.WriteWithCAS(ctx, path, data, info); err != nil {
			if errors.Is(err, ErrConcurrentModification) {
				// Retry on concurrent modification
				continue
			}
			return err
		}

		return nil
	}

	return fmt.Errorf("failed after %d retries: %w", maxRetries, ErrConcurrentModification)
}

// Delete removes a file with an exclusive lock
func (m *Manager[T]) Delete(ctx context.Context, path string) error {
	// Check if file exists first
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Create a file lock
	lock := flock.New(path)

	// Try to acquire exclusive lock with timeout
	lockCtx, cancel := context.WithTimeout(ctx, m.lockTimeout)
	defer cancel()

	locked, err := lock.TryLockContext(lockCtx, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to acquire write lock: %w", err)
	}
	if !locked {
		return ErrLockTimeout
	}

	// Unlock before removing on Windows to avoid file handle issues
	if err := lock.Unlock(); err != nil {
		return fmt.Errorf("failed to unlock file: %w", err)
	}

	// Remove the file
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	return nil
}
