package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aki/amux/internal/semaphore"
)

// sessionHolder implements semaphore.Holder interface for session IDs
type sessionHolder struct {
	sessionID string
}

func (h *sessionHolder) ID() string {
	return h.sessionID
}

// getSemaphorePath returns the path to the workspace semaphore file
func (w *Workspace) getSemaphorePath() string {
	// Semaphore is stored in the workspace directory, not in the worktree
	// Extract workspace directory from the worktree path
	workspaceDir := filepath.Dir(w.Path) // Remove "worktree" from path
	return filepath.Join(workspaceDir, "semaphore.json")
}

// Acquire acquires the workspace semaphore for a session
func (w *Workspace) Acquire(sessionID string) error {
	semaphorePath := w.getSemaphorePath()

	// Create semaphore with capacity 10 (reasonable limit for concurrent sessions)
	sem, err := semaphore.New(semaphorePath, 10)
	if err != nil {
		return fmt.Errorf("failed to create semaphore: %w", err)
	}
	defer func() {
		_ = sem.Close()
	}()

	// Acquire semaphore
	holder := &sessionHolder{sessionID: sessionID}
	if err := sem.Acquire(holder); err != nil {
		return fmt.Errorf("failed to acquire workspace: %w", err)
	}

	return nil
}

// Release releases the workspace semaphore for a session
func (w *Workspace) Release(sessionID string) error {
	semaphorePath := w.getSemaphorePath()

	// Create semaphore
	sem, err := semaphore.New(semaphorePath, 10)
	if err != nil {
		return fmt.Errorf("failed to create semaphore: %w", err)
	}
	defer func() {
		_ = sem.Close()
	}()

	// Release semaphore
	if err := sem.Release(sessionID); err != nil {
		return fmt.Errorf("failed to release workspace: %w", err)
	}

	return nil
}

// SessionIDs returns the list of session IDs currently using the workspace
func (w *Workspace) SessionIDs() ([]string, error) {
	semaphorePath := w.getSemaphorePath()

	// Check if semaphore file exists
	if _, err := os.Stat(semaphorePath); os.IsNotExist(err) {
		// No semaphore file means no sessions are using it
		return []string{}, nil
	}

	// Create semaphore
	sem, err := semaphore.New(semaphorePath, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to create semaphore: %w", err)
	}
	defer func() {
		_ = sem.Close()
	}()

	// Get holders
	return sem.Holders(), nil
}

// IsAvailable checks if the workspace has available capacity
func (w *Workspace) IsAvailable() (bool, error) {
	semaphorePath := w.getSemaphorePath()

	// Check if semaphore file exists
	if _, err := os.Stat(semaphorePath); os.IsNotExist(err) {
		// No semaphore file means it's available
		return true, nil
	}

	// Create semaphore
	sem, err := semaphore.New(semaphorePath, 10)
	if err != nil {
		return false, fmt.Errorf("failed to create semaphore: %w", err)
	}
	defer func() {
		_ = sem.Close()
	}()

	// Check if there's available capacity
	return sem.Available() > 0, nil
}

// SessionCount returns the number of active sessions using the workspace
func (w *Workspace) SessionCount() int {
	semaphorePath := w.getSemaphorePath()

	// Check if semaphore file exists
	if _, err := os.Stat(semaphorePath); os.IsNotExist(err) {
		// No semaphore file means no sessions
		return 0
	}

	// Create semaphore
	sem, err := semaphore.New(semaphorePath, 10)
	if err != nil {
		return 0
	}
	defer func() {
		_ = sem.Close()
	}()

	return sem.Count()
}
