package session

import (
	"context"
	"fmt"
	"time"

	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/session/state"
	"github.com/aki/amux/internal/core/workspace"
)

// SemaphoreHandler manages semaphore lifecycle based on state transitions.
type SemaphoreHandler struct {
	workspaceManager WorkspaceManager
	logger           logger.Logger
}

// WorkspaceManager interface for semaphore operations
type WorkspaceManager interface {
	AcquireSemaphore(workspaceID string, holder workspace.Holder) error
	ReleaseSemaphore(workspaceID string, holderID string) error
}

// NewSemaphoreHandler creates a new semaphore handler
func NewSemaphoreHandler(wsManager WorkspaceManager, logger logger.Logger) *SemaphoreHandler {
	return &SemaphoreHandler{
		workspaceManager: wsManager,
		logger:           logger,
	}
}

// HandleStateChange handles semaphore operations for state transitions
func (h *SemaphoreHandler) HandleStateChange(ctx context.Context, from, to state.Status, sessionID, workspaceID string) error {
	h.logger.Debug("semaphore handler called",
		"from", from,
		"to", to,
		"session", sessionID,
		"workspace", workspaceID)

	switch {
	case from == state.StatusCreated && to == state.StatusStarting:
		// Acquire semaphore when starting
		holder := workspace.Holder{
			ID:          sessionID,
			Type:        workspace.HolderTypeSession,
			SessionID:   sessionID,
			WorkspaceID: workspaceID,
			Timestamp:   time.Now(),
			Description: fmt.Sprintf("Session %s", sessionID),
		}

		if err := h.workspaceManager.AcquireSemaphore(workspaceID, holder); err != nil {
			h.logger.Error("failed to acquire semaphore",
				"session", sessionID,
				"workspace", workspaceID,
				"error", err)
			return fmt.Errorf("failed to acquire semaphore: %w", err)
		}

		h.logger.Debug("semaphore acquired",
			"session", sessionID,
			"workspace", workspaceID)

	case to.IsTerminal():
		// Release semaphore on any terminal state
		if err := h.workspaceManager.ReleaseSemaphore(workspaceID, sessionID); err != nil {
			h.logger.Error("failed to release semaphore",
				"session", sessionID,
				"workspace", workspaceID,
				"error", err)
			// Don't return error - we want to proceed even if release fails
		} else {
			h.logger.Debug("semaphore released",
				"session", sessionID,
				"workspace", workspaceID)
		}
	}

	return nil
}

// TmuxHandler manages tmux operations based on state transitions.
type TmuxHandler struct {
	tmuxAdapter TmuxAdapter
	logger      logger.Logger
}

// TmuxAdapter interface for tmux operations
type TmuxAdapter interface {
	CreateSession(name string, command string, workDir string, env map[string]string) error
	KillSession(name string) error
	HasSession(name string) (bool, error)
}

// NewTmuxHandler creates a new tmux handler
func NewTmuxHandler(tmux TmuxAdapter, logger logger.Logger) *TmuxHandler {
	return &TmuxHandler{
		tmuxAdapter: tmux,
		logger:      logger,
	}
}

// HandleStateChange handles tmux operations for state transitions
func (h *TmuxHandler) HandleStateChange(ctx context.Context, from, to state.Status, sessionID, workspaceID string) error {
	// This would be implemented based on specific tmux session details
	// For now, we'll leave it as a placeholder
	return nil
}
