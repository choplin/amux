package workspace

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/aki/amux/internal/core/semaphore"
)

// Session represents a session that can hold a workspace semaphore
type Session interface {
	ID() string
}

// Default semaphore capacity - one session per workspace
const defaultSemaphoreCapacity = 1

// initSemaphore initializes the semaphore for a workspace
func (m *Manager) initSemaphore(workspaceID string) *semaphore.FileSemaphore {
	// For now, use default capacity of 1
	// TODO: Make this configurable via config file
	capacity := defaultSemaphoreCapacity
	return semaphore.New(filepath.Join(m.workspacesDir, workspaceID, "semaphore.json"), capacity)
}

// AcquireWorkspace acquires a workspace for a session
func (m *Manager) AcquireWorkspace(workspaceID string, session Session) error {
	sem := m.initSemaphore(workspaceID)
	return sem.Acquire(session)
}

// ReleaseWorkspace releases a workspace held by a session
func (m *Manager) ReleaseWorkspace(workspaceID string, sessionID string) error {
	sem := m.initSemaphore(workspaceID)

	// Get current holders to find the one to remove
	holders := sem.Holders()
	for _, id := range holders {
		if id == sessionID {
			return sem.Remove(sessionID)
		}
	}

	// Not found is not an error (idempotent)
	return nil
}

// GetWorkspaceSessionIDs returns all session IDs using a workspace
func (m *Manager) GetWorkspaceSessionIDs(workspaceID string) []string {
	sem := m.initSemaphore(workspaceID)
	return sem.Holders()
}

// GetWorkspaceSessionCount returns the number of sessions using a workspace
func (m *Manager) GetWorkspaceSessionCount(workspaceID string) int {
	sem := m.initSemaphore(workspaceID)
	return sem.Count()
}

// IsWorkspaceAvailable checks if a workspace can accept new sessions
func (m *Manager) IsWorkspaceAvailable(workspaceID string) bool {
	sem := m.initSemaphore(workspaceID)
	return sem.Available() > 0
}

// RemoveWithSessionCheck removes a workspace, checking if it's in use by sessions
func (m *Manager) RemoveWithSessionCheck(ctx context.Context, identifier Identifier, force bool) error {
	workspace, err := m.ResolveWorkspace(ctx, identifier)
	if err != nil {
		return err
	}

	if !force {
		// Check if workspace is in use
		sessionIDs := m.GetWorkspaceSessionIDs(workspace.ID)
		if len(sessionIDs) > 0 {
			return fmt.Errorf("workspace %s is in use by %d session(s)", workspace.ID, len(sessionIDs))
		}
	}

	// Remove the workspace
	return m.Remove(ctx, identifier)
}

// ReconcileWorkspaceSessions removes stale session IDs from a workspace
func (m *Manager) ReconcileWorkspaceSessions(workspaceID string, isActiveFunc func(sessionID string) bool) error {
	sem := m.initSemaphore(workspaceID)
	sessionIDs := sem.Holders()

	staleSessionIDs := []string{}
	for _, sessionID := range sessionIDs {
		if !isActiveFunc(sessionID) {
			staleSessionIDs = append(staleSessionIDs, sessionID)
		}
	}

	if len(staleSessionIDs) > 0 {
		return sem.Remove(staleSessionIDs...)
	}

	return nil
}
