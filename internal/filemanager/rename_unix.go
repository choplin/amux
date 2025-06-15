//go:build !windows

package filemanager

import "os"

// atomicRename performs an atomic rename operation on Unix-like systems.
// On Unix, os.Rename is already atomic.
func atomicRename(src, dst string) error {
	return os.Rename(src, dst)
}
