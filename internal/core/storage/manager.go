// Package storage provides file storage operations for workspaces and sessions
package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Manager provides storage operations for workspaces and sessions
type Manager struct{}

// NewManager creates a new storage manager
func NewManager() *Manager {
	return &Manager{}
}

// ReadFile reads a file from the storage path
func (m *Manager) ReadFile(ctx context.Context, storagePath, relativePath string) ([]byte, error) {
	if storagePath == "" {
		return nil, fmt.Errorf("storage path not found")
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
			return nil, fmt.Errorf("file not found: %s", relativePath)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return content, nil
}

// WriteFile writes content to a file in the storage path
func (m *Manager) WriteFile(ctx context.Context, storagePath, relativePath string, content []byte) error {
	if storagePath == "" {
		return fmt.Errorf("storage path not found")
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

// Remove removes a file or directory from the storage path
func (m *Manager) Remove(ctx context.Context, storagePath, relativePath string, recursive bool) (*RemoveResult, error) {
	if storagePath == "" {
		return nil, fmt.Errorf("storage path not found")
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
			return nil, fmt.Errorf("file not found: %s", relativePath)
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

// ListFiles lists files in the storage path
func (m *Manager) ListFiles(ctx context.Context, storagePath, relativePath string) (*ListResult, error) {
	if storagePath == "" {
		return nil, fmt.Errorf("storage path not found")
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
			return nil, fmt.Errorf("path does not exist: %s", relativePath)
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
			Name:    info.Name(),
			Size:    info.Size(),
			IsDir:   false,
			ModTime: info.ModTime(),
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
		info, err := entry.Info()
		if err != nil {
			continue // Skip files we can't stat
		}
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
		})
	}

	result.Files = files
	return result, nil
}

// validatePath ensures the path is within the storage directory
func (m *Manager) validatePath(storagePath, fullPath string) error {
	cleanPath := filepath.Clean(fullPath)
	cleanStoragePath := filepath.Clean(storagePath)
	if !strings.HasPrefix(cleanPath, cleanStoragePath) {
		return fmt.Errorf("path traversal attempt detected")
	}
	return nil
}
