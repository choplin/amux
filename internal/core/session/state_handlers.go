package session

import (
	"context"
	"fmt"

	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/session/state"
	"github.com/aki/amux/internal/core/workspace"
)

// SemaphoreHandler manages semaphore lifecycle based on state transitions.
type SemaphoreHandler struct {
	workspaceManager WorkspaceManager
	sessionResolver  SessionResolver
	logger           logger.Logger
}

// WorkspaceManager interface for workspace session management
type WorkspaceManager interface {
	AcquireWorkspace(workspaceID string, session workspace.Session) error
	ReleaseWorkspace(workspaceID string, sessionID string) error
}

// SessionResolver interface to get session objects
type SessionResolver interface {
	Get(ctx context.Context, sessionID string) (Session, error)
}

// NewSemaphoreHandler creates a new semaphore handler
func NewSemaphoreHandler(wsManager WorkspaceManager, sessionResolver SessionResolver, logger logger.Logger) *SemaphoreHandler {
	return &SemaphoreHandler{
		workspaceManager: wsManager,
		sessionResolver:  sessionResolver,
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
		// Acquire workspace when starting
		session, err := h.sessionResolver.Get(ctx, sessionID)
		if err != nil {
			h.logger.Error("failed to get session",
				"session", sessionID,
				"error", err)
			return fmt.Errorf("failed to get session: %w", err)
		}

		if err := h.workspaceManager.AcquireWorkspace(workspaceID, session); err != nil {
			h.logger.Error("failed to acquire workspace",
				"session", sessionID,
				"workspace", workspaceID,
				"error", err)
			return fmt.Errorf("failed to acquire workspace: %w", err)
		}

		h.logger.Debug("workspace acquired",
			"session", sessionID,
			"workspace", workspaceID)

	case to.IsTerminal():
		// Release workspace on any terminal state
		if err := h.workspaceManager.ReleaseWorkspace(workspaceID, sessionID); err != nil {
			h.logger.Error("failed to release workspace",
				"session", sessionID,
				"workspace", workspaceID,
				"error", err)
			// Don't return error - we want to proceed even if release fails
		} else {
			h.logger.Debug("workspace released",
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
