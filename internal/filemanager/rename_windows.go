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
			// ERROR_SHARING_VIOLATION = 32
			if errno == 5 || errno == 183 || errno == 32 {
				// Try to remove the destination file first with retries
				for i := 0; i < 3; i++ {
					removeErr := os.Remove(dst)
					if removeErr == nil {
						// Successfully removed, now try rename
						time.Sleep(10 * time.Millisecond)
						return os.Rename(src, dst)
					}
					// If it's not exist error, wait and retry
					if os.IsNotExist(removeErr) {
						// File doesn't exist, try rename directly
						return os.Rename(src, dst)
					}
					// Wait longer with each retry
					time.Sleep(time.Duration(20*(i+1)) * time.Millisecond)
				}
				// Final attempt after all retries
				return os.Rename(src, dst)
			}
		}
	}

	return err
}
