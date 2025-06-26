// Package semaphore provides a file-based semaphore implementation for
// coordinating access to shared resources across multiple processes.
// See ADR-032 for design decisions.
package semaphore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Holder represents an entity that can hold a semaphore
type Holder interface {
	ID() string
}

// holderEntry represents a holder with metadata
type holderEntry struct {
	ID         string    `json:"id"`
	AcquiredAt time.Time `json:"acquired_at"`
}

// semaphoreData represents the persistent state of a semaphore
type semaphoreData struct {
	Capacity int           `json:"capacity"`
	Holders  []holderEntry `json:"holders"`
}

// FileSemaphore manages access to a resource using file-based locking
type FileSemaphore struct {
	path     string
	capacity int
	mu       sync.Mutex
	lock     *fileLock
}

// Error types
var (
	ErrNoCapacity  = errors.New("semaphore has no available capacity")
	ErrAlreadyHeld = errors.New("semaphore already held by this holder")
	ErrNotHeld     = errors.New("semaphore not held by this holder")
)

// New creates a new file-based semaphore
func New(path string, capacity int) (*FileSemaphore, error) {
	if capacity < 1 {
		capacity = 1
	}

	// Create file lock
	lock, err := newFileLock(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create file lock: %w", err)
	}

	return &FileSemaphore{
		path:     path,
		capacity: capacity,
		lock:     lock,
	}, nil
}

// Acquire attempts to acquire the semaphore for a holder
func (s *FileSemaphore) Acquire(holder Holder) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire file lock for cross-process synchronization
	if err := s.lock.lock(); err != nil {
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	defer func() {
		_ = s.lock.unlock()
	}()

	data, err := s.load()
	if err != nil {
		return fmt.Errorf("failed to load semaphore data: %w", err)
	}

	// Check if already held
	holderID := holder.ID()
	for _, h := range data.Holders {
		if h.ID == holderID {
			return ErrAlreadyHeld
		}
	}

	// Check capacity
	if len(data.Holders) >= s.capacity {
		return ErrNoCapacity
	}

	// Add holder
	data.Holders = append(data.Holders, holderEntry{
		ID:         holderID,
		AcquiredAt: time.Now(),
	})

	return s.save(data)
}

// Release releases the semaphore for a specific holder ID
func (s *FileSemaphore) Release(holderID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire file lock for cross-process synchronization
	if err := s.lock.lock(); err != nil {
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	defer func() {
		_ = s.lock.unlock()
	}()

	data, err := s.load()
	if err != nil {
		return fmt.Errorf("failed to load semaphore data: %w", err)
	}

	// Find and remove holder
	found := false
	filtered := make([]holderEntry, 0, len(data.Holders))
	for _, h := range data.Holders {
		if h.ID == holderID {
			found = true
			continue
		}
		filtered = append(filtered, h)
	}

	if !found {
		return ErrNotHeld
	}

	data.Holders = filtered
	return s.save(data)
}

// Remove removes one or more holders from the semaphore
func (s *FileSemaphore) Remove(holderIDs ...string) error {
	if len(holderIDs) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire file lock for cross-process synchronization
	if err := s.lock.lock(); err != nil {
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	defer func() {
		_ = s.lock.unlock()
	}()

	data, err := s.load()
	if err != nil {
		return fmt.Errorf("failed to load semaphore data: %w", err)
	}

	// Create a set for fast lookup
	toRemove := make(map[string]bool)
	for _, id := range holderIDs {
		toRemove[id] = true
	}

	// Filter out holders
	filtered := make([]holderEntry, 0, len(data.Holders))
	for _, h := range data.Holders {
		if !toRemove[h.ID] {
			filtered = append(filtered, h)
		}
	}

	data.Holders = filtered
	return s.save(data)
}

// Holders returns the current holder IDs
func (s *FileSemaphore) Holders() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire file lock for cross-process synchronization
	if err := s.lock.lock(); err != nil {
		return nil
	}
	defer func() {
		_ = s.lock.unlock()
	}()

	data, err := s.load()
	if err != nil {
		return nil
	}

	ids := make([]string, len(data.Holders))
	for i, h := range data.Holders {
		ids[i] = h.ID
	}
	return ids
}

// Count returns the number of current holders
func (s *FileSemaphore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire file lock for cross-process synchronization
	if err := s.lock.lock(); err != nil {
		return 0
	}
	defer func() {
		_ = s.lock.unlock()
	}()

	data, err := s.load()
	if err != nil {
		return 0
	}

	return len(data.Holders)
}

// Available returns the number of available slots
func (s *FileSemaphore) Available() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Acquire file lock for cross-process synchronization
	if err := s.lock.lock(); err != nil {
		return s.capacity
	}
	defer func() {
		_ = s.lock.unlock()
	}()

	data, err := s.load()
	if err != nil {
		return s.capacity
	}

	available := s.capacity - len(data.Holders)
	if available < 0 {
		return 0
	}
	return available
}

// Close closes the semaphore and releases resources
func (s *FileSemaphore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lock != nil {
		return s.lock.close()
	}
	return nil
}

// load loads the semaphore data from disk
func (s *FileSemaphore) load() (*semaphoreData, error) {
	data := &semaphoreData{
		Capacity: s.capacity,
		Holders:  []holderEntry{},
	}

	// If file doesn't exist, return empty data
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return data, nil
	}

	content, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(content, data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Update capacity if it changed
	data.Capacity = s.capacity

	return data, nil
}

// save saves the semaphore data to disk atomically
func (s *FileSemaphore) save(data *semaphoreData) error {
	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal data
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to temporary file
	tempFile := s.path + ".tmp"
	if err := os.WriteFile(tempFile, content, 0o644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, s.path); err != nil {
		_ = os.Remove(tempFile) // Clean up on failure
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}
