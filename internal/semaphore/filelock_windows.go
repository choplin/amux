//go:build windows

package semaphore

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	// Windows lock flags
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
)

// fileLock provides file locking functionality using Windows LockFileEx
type fileLock struct {
	file *os.File
}

// newFileLock creates a new file lock
func newFileLock(path string) (*fileLock, error) {
	lockPath := path + ".lock"

	// Ensure directory exists
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Open or create the lock file
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}
	return &fileLock{file: file}, nil
}

// lock acquires an exclusive lock on the file
func (fl *fileLock) lock() error {
	handle := fl.file.Fd()
	var overlapped syscall.Overlapped

	// LockFileEx parameters:
	// - handle: file handle
	// - flags: LOCKFILE_EXCLUSIVE_LOCK | LOCKFILE_FAIL_IMMEDIATELY
	// - reserved: must be 0
	// - numberOfBytesToLockLow: low-order 32 bits of length to lock
	// - numberOfBytesToLockHigh: high-order 32 bits of length to lock
	// - overlapped: pointer to OVERLAPPED structure
	ret, _, err := procLockFileEx.Call(
		uintptr(handle),
		uintptr(lockfileExclusiveLock|lockfileFailImmediately),
		0, // reserved
		1, // lock 1 byte
		0, // high dword of 1 byte
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if ret == 0 {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	return nil
}

// unlock releases the lock on the file
func (fl *fileLock) unlock() error {
	handle := fl.file.Fd()
	var overlapped syscall.Overlapped

	// UnlockFileEx parameters:
	// - handle: file handle
	// - reserved: must be 0
	// - numberOfBytesToUnlockLow: low-order 32 bits of length to unlock
	// - numberOfBytesToUnlockHigh: high-order 32 bits of length to unlock
	// - overlapped: pointer to OVERLAPPED structure
	ret, _, err := procUnlockFileEx.Call(
		uintptr(handle),
		0, // reserved
		1, // unlock 1 byte
		0, // high dword of 1 byte
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if ret == 0 {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}

// close closes the file
func (fl *fileLock) close() error {
	return fl.file.Close()
}
