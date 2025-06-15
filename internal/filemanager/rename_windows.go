//go:build windows

package filemanager

import (
	"os"
	"syscall"
	"time"
)

// atomicRename performs an atomic rename operation on Windows.
// On Windows, we need to handle the case where the destination file already exists.
func atomicRename(src, dst string) error {
	// Try direct rename first
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// If that fails, check if it's a permission error or file exists error
	if linkErr, ok := err.(*os.LinkError); ok {
		if errno, ok := linkErr.Err.(syscall.Errno); ok {
			// ERROR_ACCESS_DENIED = 5
			// ERROR_ALREADY_EXISTS = 183
			if errno == 5 || errno == 183 {
				// Try to remove the destination file first
				_ = os.Remove(dst)
				// Small delay to ensure Windows releases the file handle
				time.Sleep(10 * time.Millisecond)
				// Try rename again
				return os.Rename(src, dst)
			}
		}
	}

	return err
}
