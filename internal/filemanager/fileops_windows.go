//go:build windows

package filemanager

import (
	"math/rand"
	"os"
	"strings"
	"time"
)

// readFileWithRetry attempts to read a file with retries on Windows
// to handle temporary file locking issues.
func readFileWithRetry(path string) ([]byte, error) {
	var data []byte
	var err error

	// Add a small random delay to reduce contention
	time.Sleep(time.Duration(rand.Intn(5)) * time.Millisecond)

	for i := 0; i < 5; i++ {
		data, err = os.ReadFile(path)
		if err == nil {
			return data, nil
		}

		// If it's a sharing violation, wait and retry with exponential backoff
		if os.IsPermission(err) || isFileLocked(err) {
			delay := time.Duration(10*(1<<uint(i))) * time.Millisecond
			// Add jitter to prevent thundering herd
			jitter := time.Duration(rand.Intn(10)) * time.Millisecond
			time.Sleep(delay + jitter)
			continue
		}

		// For other errors, return immediately
		return nil, err
	}

	return nil, err
}

// isFileLocked checks if the error is due to file being locked
func isFileLocked(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for common Windows file locking error messages
	return strings.Contains(errStr, "being used by another process") ||
		strings.Contains(errStr, "The process cannot access")
}
