// Package semaphore provides a file-based semaphore implementation for resource limiting.
package semaphore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Holder represents an entity that can acquire a semaphore.
type Holder interface {
	ID() string
}

// FileSemaphore implements a file-based counting semaphore.
type FileSemaphore struct {
	path     string
	capacity int
	mu       sync.Mutex
}

// New creates a new file-based semaphore with the given capacity.
func New(path string, capacity int) *FileSemaphore {
	if capacity < 1 {
		capacity = 1
	}
	return &FileSemaphore{
		path:     path,
		capacity: capacity,
	}
}

// semaphoreData represents the persistent state of the semaphore.
type semaphoreData struct {
	Capacity int      `json:"capacity"`
	Holders  []string `json:"holders"`
}

// Acquire attempts to acquire the semaphore for the given holder.
// Returns ErrSemaphoreFull if the semaphore is at capacity.
// Returns ErrAlreadyHolder if the holder already has the semaphore.
func (s *FileSemaphore) Acquire(holder Holder) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return fmt.Errorf("failed to load semaphore: %w", err)
	}

	holderID := holder.ID()

	// Check if already holding
	for _, h := range data.Holders {
		if h == holderID {
			return ErrAlreadyHolder{ID: holderID}
		}
	}

	// Check capacity
	if len(data.Holders) >= s.capacity {
		return ErrSemaphoreFull{
			Capacity: s.capacity,
			Current:  len(data.Holders),
		}
	}

	// Acquire
	data.Holders = append(data.Holders, holderID)
	data.Capacity = s.capacity

	if err := s.save(data); err != nil {
		return fmt.Errorf("failed to save semaphore: %w", err)
	}

	return nil
}

// Release releases the semaphore for the given holder.
// Returns ErrNotHolder if the holder does not have the semaphore.
func (s *FileSemaphore) Release(holder Holder) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return fmt.Errorf("failed to load semaphore: %w", err)
	}

	holderID := holder.ID()
	found := false
	newHolders := make([]string, 0, len(data.Holders))

	for _, h := range data.Holders {
		if h == holderID {
			found = true
			continue
		}
		newHolders = append(newHolders, h)
	}

	if !found {
		return ErrNotHolder{ID: holderID}
	}

	data.Holders = newHolders
	data.Capacity = s.capacity

	if err := s.save(data); err != nil {
		return fmt.Errorf("failed to save semaphore: %w", err)
	}

	return nil
}

// IsHeld returns true if the given holder has acquired the semaphore.
func (s *FileSemaphore) IsHeld(holder Holder) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return false
	}

	holderID := holder.ID()
	for _, h := range data.Holders {
		if h == holderID {
			return true
		}
	}

	return false
}

// Holders returns the IDs of all current holders.
func (s *FileSemaphore) Holders() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return nil
	}

	// Return a copy to prevent external modification
	holders := make([]string, len(data.Holders))
	copy(holders, data.Holders)
	return holders
}

// Count returns the number of current holders.
func (s *FileSemaphore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return 0
	}

	return len(data.Holders)
}

// Available returns the number of available slots.
func (s *FileSemaphore) Available() int {
	count := s.Count()
	available := s.capacity - count
	if available < 0 {
		return 0
	}
	return available
}

// Capacity returns the total capacity of the semaphore.
func (s *FileSemaphore) Capacity() int {
	return s.capacity
}

// Clear removes all holders from the semaphore.
func (s *FileSemaphore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := &semaphoreData{
		Capacity: s.capacity,
		Holders:  []string{},
	}

	if err := s.save(data); err != nil {
		return fmt.Errorf("failed to clear semaphore: %w", err)
	}

	return nil
}

// Remove removes specific holders from the semaphore.
// It's idempotent - removing non-existent holders is not an error.
func (s *FileSemaphore) Remove(holderIDs ...string) error {
	if len(holderIDs) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.load()
	if err != nil {
		return fmt.Errorf("failed to load semaphore: %w", err)
	}

	// Create a set for efficient lookup
	toRemove := make(map[string]bool)
	for _, id := range holderIDs {
		toRemove[id] = true
	}

	// Filter out holders to be removed
	newHolders := make([]string, 0, len(data.Holders))
	for _, h := range data.Holders {
		if !toRemove[h] {
			newHolders = append(newHolders, h)
		}
	}

	data.Holders = newHolders
	data.Capacity = s.capacity

	if err := s.save(data); err != nil {
		return fmt.Errorf("failed to save semaphore: %w", err)
	}

	return nil
}

// load reads the semaphore data from disk.
func (s *FileSemaphore) load() (*semaphoreData, error) {
	// If file doesn't exist, return empty semaphore
	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return &semaphoreData{
			Capacity: s.capacity,
			Holders:  []string{},
		}, nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read semaphore file: %w", err)
	}

	var sd semaphoreData
	if err := json.Unmarshal(data, &sd); err != nil {
		return nil, fmt.Errorf("failed to unmarshal semaphore data: %w", err)
	}

	// Update capacity if it changed
	sd.Capacity = s.capacity

	return &sd, nil
}

// save writes the semaphore data to disk atomically.
func (s *FileSemaphore) save(data *semaphoreData) error {
	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create semaphore directory: %w", err)
	}

	// Marshal data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal semaphore data: %w", err)
	}

	// Write to temp file
	tempFile := s.path + ".tmp"
	if err := os.WriteFile(tempFile, jsonData, 0o644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, s.path); err != nil {
		os.Remove(tempFile) // Clean up on failure
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}
