package storage

import "time"

// FileInfo represents information about a file or directory
type FileInfo struct {
	Name    string
	Size    int64
	IsDir   bool
	ModTime time.Time
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
