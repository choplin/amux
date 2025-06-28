package session

import (
	"context"
	"log/slog"

	"github.com/aki/amux/internal/core/workspace"
)

// ReconcileWorkspaceSemaphores reconciles workspace semaphores with active sessions
// This should be called on startup to clean up stale semaphore entries
func (m *Manager) ReconcileWorkspaceSemaphores(ctx context.Context) error {
	// Skip if no workspace manager
	if m.workspaceManager == nil {
		return nil
	}

	// Get all workspaces
	workspaces, err := m.workspaceManager.List(ctx, workspace.ListOptions{})
	if err != nil {
		return err
	}

	// Get all sessions
	sessions, err := m.ListSessions(ctx)
	if err != nil {
		return err
	}

	// Build a map of active session IDs for quick lookup
	activeSessionIDs := make(map[string]bool)
	for _, sess := range sessions {
		// Only running sessions should hold semaphores
		if sess.Status().IsRunning() {
			activeSessionIDs[sess.ID()] = true
		}
	}

	// Check each workspace's semaphore
	for _, ws := range workspaces {
		// Get session IDs holding this workspace
		sessionIDs, err := ws.SessionIDs()
		if err != nil {
			slog.Warn("failed to get session IDs for workspace",
				"workspace_id", ws.ID,
				"error", err)
			continue
		}

		// Check for stale entries
		var staleIDs []string
		for _, sessionID := range sessionIDs {
			if !activeSessionIDs[sessionID] {
				staleIDs = append(staleIDs, sessionID)
			}
		}

		// Clean up stale entries
		if len(staleIDs) > 0 {
			for _, sessionID := range staleIDs {
				if err := ws.Release(sessionID); err != nil {
					slog.Warn("failed to release stale semaphore",
						"workspace_id", ws.ID,
						"session_id", sessionID,
						"error", err)
				} else {
					slog.Info("released stale workspace semaphore",
						"workspace_id", ws.ID,
						"session_id", sessionID)
				}
			}
		}
	}

	return nil
}
