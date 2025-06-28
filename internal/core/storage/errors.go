package storage

import "fmt"

// ErrNotFound is returned when a file or path is not found
type ErrNotFound struct {
	Path string
}

func (e ErrNotFound) Error() string {
	if e.Path == "" {
		return "path does not exist"
	}
	return fmt.Sprintf("path does not exist: %s", e.Path)
}

// ErrNotDirectory is returned when attempting directory operations on a file
type ErrNotDirectory struct {
	Path string
}

func (e ErrNotDirectory) Error() string {
	return fmt.Sprintf("not a directory: %s", e.Path)
}

// ErrPathTraversal is returned when a path traversal attempt is detected
type ErrPathTraversal struct{}

func (e ErrPathTraversal) Error() string {
	return "path traversal attempt detected"
}

// ErrStoragePathEmpty is returned when storage path is empty
type ErrStoragePathEmpty struct{}

func (e ErrStoragePathEmpty) Error() string {
	return "storage path not found"
}
