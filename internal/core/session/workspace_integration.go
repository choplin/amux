package session

import (
	"context"

	"github.com/aki/amux/internal/core/workspace"
)

// WorkspaceSessionChecker adapts session.Manager to workspace.SessionChecker interface
type WorkspaceSessionChecker struct {
	sessionManager *Manager
}

// NewWorkspaceSessionChecker creates a new adapter for session checking
func NewWorkspaceSessionChecker(sessionManager *Manager) workspace.SessionChecker {
	return &WorkspaceSessionChecker{
		sessionManager: sessionManager,
	}
}

// IsSessionActive checks if a session exists and is active
func (w *WorkspaceSessionChecker) IsSessionActive(sessionID string) (bool, error) {
	sess, err := w.sessionManager.Get(context.Background(), ID(sessionID))
	if err != nil {
		// Session not found = not active
		return false, nil
	}

	// Check if session is in a running state
	status := sess.Status()
	return status.IsRunning(), nil
}
