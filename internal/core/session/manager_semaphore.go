package session

import (
	"context"
	"fmt"
	"time"

	"github.com/aki/amux/internal/core/workspace"
)

// SessionStopperAdapter implements workspace.SessionStopper interface
type SessionStopperAdapter struct {
	manager *Manager
}

// NewSessionStopperAdapter creates a new adapter
func NewSessionStopperAdapter(manager *Manager) *SessionStopperAdapter {
	return &SessionStopperAdapter{manager: manager}
}

// StopSessionsInWorkspace stops all sessions in a workspace
func (s *SessionStopperAdapter) StopSessionsInWorkspace(ctx context.Context, workspaceID string) error {
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
func (s *SessionStopperAdapter) ListSessionsInWorkspace(ctx context.Context, workspaceID string) ([]string, error) {
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

// acquireSemaphore acquires a workspace semaphore for a session
func (m *Manager) acquireSemaphore(ctx context.Context, info *Info) error {
	if m.workspaceManager == nil {
		// No workspace manager, skip semaphore
		return nil
	}

	holder := workspace.Holder{
		ID:          info.ID,
		Type:        workspace.HolderTypeSession,
		SessionID:   info.ID,
		WorkspaceID: info.WorkspaceID,
		Timestamp:   time.Now(),
		Description: fmt.Sprintf("%s: %s", info.AgentID, info.Description),
	}

	if holder.Description == info.AgentID+":" || holder.Description == info.AgentID+": " {
		// Use command if no description
		holder.Description = fmt.Sprintf("%s: %s", info.AgentID, info.Command)
	}

	return m.workspaceManager.AcquireSemaphore(info.WorkspaceID, holder)
}

// releaseSemaphore releases a workspace semaphore for a session
func (m *Manager) releaseSemaphore(ctx context.Context, sessionID, workspaceID string) error {
	if m.workspaceManager == nil {
		// No workspace manager, skip semaphore
		return nil
	}

	return m.workspaceManager.ReleaseSemaphore(workspaceID, sessionID)
}
