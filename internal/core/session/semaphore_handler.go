package session

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aki/amux/internal/core/session/state"
	"github.com/aki/amux/internal/core/workspace"
)

// createSemaphoreHandler creates a state change handler that manages workspace semaphore acquisition/release
func createSemaphoreHandler(workspaceManager *workspace.Manager) state.ChangeHandler {
	return func(ctx context.Context, from, to state.Status, sessionID, workspaceID string) error {
		// Skip if no workspace ID
		if workspaceID == "" {
			return nil
		}

		// Get workspace
		ws, err := workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(workspaceID))
		if err != nil {
			// If workspace not found, log but don't fail the state transition
			slog.Warn("workspace not found for semaphore operation",
				"workspace_id", workspaceID,
				"session_id", sessionID,
				"error", err)
			return nil
		}

		// Handle acquisition when transitioning to running
		if from == state.StatusStarting && to == state.StatusRunning {
			if err := ws.Acquire(sessionID); err != nil {
				return fmt.Errorf("failed to acquire workspace semaphore: %w", err)
			}
			slog.Debug("acquired workspace semaphore",
				"workspace_id", workspaceID,
				"session_id", sessionID)
		}

		// Handle release when transitioning to terminal states
		if !from.IsTerminal() && to.IsTerminal() {
			if err := ws.Release(sessionID); err != nil {
				// Log error but don't fail the state transition
				slog.Warn("failed to release workspace semaphore",
					"workspace_id", workspaceID,
					"session_id", sessionID,
					"error", err)
			} else {
				slog.Debug("released workspace semaphore",
					"workspace_id", workspaceID,
					"session_id", sessionID)
			}
		}

		return nil
	}
}
