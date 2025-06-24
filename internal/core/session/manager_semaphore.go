package session

import (
	"context"
	"fmt"
)

// StopperAdapter implements workspace.SessionStopper interface.
type StopperAdapter struct {
	manager *Manager
}

// NewSessionStopperAdapter creates a new adapter
func NewSessionStopperAdapter(manager *Manager) *StopperAdapter {
	return &StopperAdapter{manager: manager}
}

// StopSessionsInWorkspace stops all sessions in a workspace
func (s *StopperAdapter) StopSessionsInWorkspace(ctx context.Context, workspaceID string) error {
	sessions, err := s.manager.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	var errors []error
	for _, sess := range sessions {
		if err := sess.Stop(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop session %s: %w", sess.ID(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to stop %d sessions", len(errors))
	}

	return nil
}

// ListSessionsInWorkspace lists all sessions in a workspace
func (s *StopperAdapter) ListSessionsInWorkspace(ctx context.Context, workspaceID string) ([]string, error) {
	sessions, err := s.manager.ListByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(sessions))
	for i, sess := range sessions {
		ids[i] = string(sess.ID())
	}

	return ids, nil
}

// releaseSemaphore releases a workspace semaphore for a session
func (m *Manager) releaseSemaphore(ctx context.Context, sessionID, workspaceID string) error {
	if m.workspaceManager == nil {
		// No workspace manager, skip semaphore
		return nil
	}

	return m.workspaceManager.ReleaseWorkspace(workspaceID, sessionID)
}
