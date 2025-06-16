//go:build !windows

package filemanager

import "os"

// readFileWithRetry on Unix just delegates to os.ReadFile
// since file locking issues are less common on Unix systems.
func readFileWithRetry(path string) ([]byte, error) {
	return os.ReadFile(path)
}
