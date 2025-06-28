package storage

import "time"

// Provider is an interface for entities that have storage
type Provider interface {
	GetStoragePath() string
}

// FileInfo represents information about a file or directory
type FileInfo struct {
	Name       string
	Size       int64
	IsDir      bool
	IsSymlink  bool
	LinkTarget string // Only set if IsSymlink is true
	ModTime    time.Time
}

// ListResult represents the result of a list operation
type ListResult struct {
	Files        []FileInfo
	IsTargetFile bool // true if the target path itself is a file (not a directory)
}

// RemoveResult represents the result of a remove operation
type RemoveResult struct {
	Path  string
	IsDir bool
}
