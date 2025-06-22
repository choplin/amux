package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

const (
	semaphoreVersion  = "1.0"
	semaphoreFileName = ".semaphore"
	lockFileName      = ".semaphore.lock"
)

// SemaphoreData represents the semaphore file contents
type SemaphoreData struct {
	Version string   `json:"version"`
	Holders []Holder `json:"holders"`
}

// SessionChecker is an interface for checking session status
type SessionChecker interface {
	// IsSessionActive checks if a session exists and is active
	IsSessionActive(sessionID string) (bool, error)
}

// SemaphoreManager manages workspace semaphores
type SemaphoreManager struct {
	basePath       string
	sessionChecker SessionChecker
	reconciler     *SemaphoreReconciler
}

// NewSemaphoreManager creates a new semaphore manager
func NewSemaphoreManager(basePath string, sessionChecker SessionChecker) *SemaphoreManager {
	sm := &SemaphoreManager{
		basePath:       basePath,
		sessionChecker: sessionChecker,
	}
	// Create reconciler after manager is initialized
	// This will be set via SetReconciler to avoid circular dependency
	return sm
}

// SetReconciler sets the reconciler for the semaphore manager
func (s *SemaphoreManager) SetReconciler(reconciler *SemaphoreReconciler) {
	s.reconciler = reconciler
}

// Acquire acquires a semaphore for a workspace
func (s *SemaphoreManager) Acquire(workspaceID string, holder Holder) error {
	// Reconcile before acquiring
	if s.reconciler != nil {
		_ = s.reconciler.ReconcileOnAcquire(workspaceID)
	}

	return s.updateSemaphore(workspaceID, func(data *SemaphoreData) error {
		// Set timestamp if not already set
		if holder.Timestamp.IsZero() {
			holder.Timestamp = time.Now()
		}

		// Ensure workspace ID matches
		holder.WorkspaceID = workspaceID

		// Add the new holder
		data.Holders = append(data.Holders, holder)
		return nil
	})
}

// Release releases a semaphore for a specific holder
func (s *SemaphoreManager) Release(workspaceID, holderID string) error {
	return s.updateSemaphore(workspaceID, func(data *SemaphoreData) error {
		// Filter out the holder with matching ID
		filtered := make([]Holder, 0, len(data.Holders))
		for _, h := range data.Holders {
			if h.ID != holderID {
				filtered = append(filtered, h)
			}
		}
		data.Holders = filtered
		return nil
	})
}

// GetHolders returns all holders for a workspace
func (s *SemaphoreManager) GetHolders(workspaceID string) ([]Holder, error) {
	data, err := s.readSemaphoreData(workspaceID)
	if err != nil {
		if os.IsNotExist(err) {
			// No semaphore file means no holders
			return []Holder{}, nil
		}
		return nil, err
	}
	return data.Holders, nil
}

// IsInUse checks if a workspace is in use by any holders
func (s *SemaphoreManager) IsInUse(workspaceID string) (bool, []Holder, error) {
	holders, err := s.GetHolders(workspaceID)
	if err != nil {
		return false, nil, err
	}
	return len(holders) > 0, holders, nil
}

// updateSemaphore performs an atomic update on the semaphore file
func (s *SemaphoreManager) updateSemaphore(workspaceID string, updater func(*SemaphoreData) error) error {
	lockPath := s.getLockPath(workspaceID)
	semaphorePath := s.getSemaphorePath(workspaceID)
	tempPath := semaphorePath + ".tmp"

	// Ensure directory exists
	dir := filepath.Dir(semaphorePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create semaphore directory: %w", err)
	}

	// Acquire file lock
	lock := flock.New(lockPath)

	// Use Lock() instead of TryLock() to wait for lock
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer func() {
		_ = lock.Unlock()
	}()

	// Read current data
	data, err := s.readSemaphoreDataLocked(semaphorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// If file doesn't exist, create new data
	if data == nil {
		data = &SemaphoreData{
			Version: semaphoreVersion,
			Holders: []Holder{},
		}
	}

	// Apply the update
	if err := updater(data); err != nil {
		return err
	}

	// Write to temporary file
	if err := s.writeToFile(tempPath, data); err != nil {
		return err
	}

	// Atomic rename
	if err := os.Rename(tempPath, semaphorePath); err != nil {
		// Clean up temp file on error
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to update semaphore file: %w", err)
	}

	return nil
}

// readSemaphoreData reads semaphore data without locking
func (s *SemaphoreManager) readSemaphoreData(workspaceID string) (*SemaphoreData, error) {
	semaphorePath := s.getSemaphorePath(workspaceID)
	return s.readSemaphoreDataLocked(semaphorePath)
}

// readSemaphoreDataLocked reads semaphore data (assumes lock is held)
func (s *SemaphoreManager) readSemaphoreDataLocked(semaphorePath string) (*SemaphoreData, error) {
	content, err := os.ReadFile(semaphorePath)
	if err != nil {
		return nil, err
	}

	var data SemaphoreData
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, fmt.Errorf("failed to parse semaphore file: %w", err)
	}

	return &data, nil
}

// writeToFile writes semaphore data to a file
func (s *SemaphoreManager) writeToFile(path string, data *SemaphoreData) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal semaphore data: %w", err)
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("failed to write semaphore file: %w", err)
	}

	return nil
}

// getSemaphorePath returns the path to the semaphore file
func (s *SemaphoreManager) getSemaphorePath(workspaceID string) string {
	return filepath.Join(s.basePath, workspaceID, semaphoreFileName)
}

// getLockPath returns the path to the lock file
func (s *SemaphoreManager) getLockPath(workspaceID string) string {
	return filepath.Join(s.basePath, workspaceID, lockFileName)
}
