// Package storage provides file storage operations for workspaces and sessions
package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Manager provides storage operations for entities with storage
type Manager[T Provider] struct {
	entity T
}

// NewManager creates a new storage manager for the given entity
func NewManager[T Provider](entity T) *Manager[T] {
	return &Manager[T]{
		entity: entity,
	}
}

// ReadFile reads a file from the entity's storage path
func (m *Manager[T]) ReadFile(ctx context.Context, relativePath string) ([]byte, error) {
	storagePath := m.entity.GetStoragePath()
	if storagePath == "" {
		return nil, ErrStoragePathEmpty{}
	}

	// Construct full path
	fullPath := filepath.Join(storagePath, relativePath)

	// Ensure the path is within the storage directory
	if err := m.validatePath(storagePath, fullPath); err != nil {
		return nil, err
	}

	// Read the file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound{Path: relativePath}
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return content, nil
}

// WriteFile writes content to a file in the entity's storage path
func (m *Manager[T]) WriteFile(ctx context.Context, relativePath string, content []byte) error {
	storagePath := m.entity.GetStoragePath()
	if storagePath == "" {
		return ErrStoragePathEmpty{}
	}

	// Construct full path
	fullPath := filepath.Join(storagePath, relativePath)

	// Ensure the path is within the storage directory
	if err := m.validatePath(storagePath, fullPath); err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Remove removes a file or directory from the entity's storage path
func (m *Manager[T]) Remove(ctx context.Context, relativePath string, recursive bool) (*RemoveResult, error) {
	storagePath := m.entity.GetStoragePath()
	if storagePath == "" {
		return nil, ErrStoragePathEmpty{}
	}

	// Construct full path
	fullPath := filepath.Join(storagePath, relativePath)

	// Ensure the path is within the storage directory
	if err := m.validatePath(storagePath, fullPath); err != nil {
		return nil, err
	}

	// Check if the path exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound{Path: relativePath}
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	result := &RemoveResult{
		Path:  relativePath,
		IsDir: info.IsDir(),
	}

	// Handle directories
	if info.IsDir() {
		if !recursive {
			return nil, fmt.Errorf("cannot remove directory without -r flag: %s", relativePath)
		}
		if err := os.RemoveAll(fullPath); err != nil {
			return nil, fmt.Errorf("failed to remove directory: %w", err)
		}
	} else {
		// Handle files
		if err := os.Remove(fullPath); err != nil {
			return nil, fmt.Errorf("failed to remove file: %w", err)
		}
	}

	return result, nil
}

// ListFiles lists files in the entity's storage path
func (m *Manager[T]) ListFiles(ctx context.Context, relativePath string) (*ListResult, error) {
	storagePath := m.entity.GetStoragePath()
	if storagePath == "" {
		return nil, ErrStoragePathEmpty{}
	}

	// Construct full path
	fullPath := filepath.Join(storagePath, relativePath)

	// Ensure the path is within the storage directory
	if err := m.validatePath(storagePath, fullPath); err != nil {
		return nil, err
	}

	// Check if the path exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound{Path: relativePath}
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	result := &ListResult{
		Files:        []FileInfo{},
		IsTargetFile: false,
	}

	// If it's a file, return file info
	if !info.IsDir() {
		result.IsTargetFile = true
		result.Files = []FileInfo{{
			Name:      info.Name(),
			Size:      info.Size(),
			IsDir:     false,
			IsSymlink: false,
			ModTime:   info.ModTime(),
		}}
		return result, nil
	}

	// List directory contents
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		entryPath := filepath.Join(fullPath, entry.Name())

		// Get entry info (doesn't follow symlinks)
		entryInfo, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}

		// Check if it's a symlink
		isSymlink := entryInfo.Mode()&os.ModeSymlink != 0
		var linkTarget string
		var targetInfo os.FileInfo

		if isSymlink {
			// Read the symlink target
			linkTarget, _ = os.Readlink(entryPath)
			// Get target info (follows symlinks)
			targetInfo, err = os.Stat(entryPath)
			if err != nil {
				// Broken symlink, use entry info
				targetInfo = entryInfo
			}
		} else {
			targetInfo = entryInfo
		}

		files = append(files, FileInfo{
			Name:       entry.Name(),
			Size:       targetInfo.Size(),
			IsDir:      targetInfo.IsDir(),
			IsSymlink:  isSymlink,
			LinkTarget: linkTarget,
			ModTime:    targetInfo.ModTime(),
		})
	}

	result.Files = files
	return result, nil
}

// validatePath ensures the path is within the storage directory
func (m *Manager[T]) validatePath(storagePath, fullPath string) error {
	cleanPath := filepath.Clean(fullPath)
	cleanStoragePath := filepath.Clean(storagePath)
	if !strings.HasPrefix(cleanPath, cleanStoragePath) {
		return ErrPathTraversal{}
	}
	return nil
}
