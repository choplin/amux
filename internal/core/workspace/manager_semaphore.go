package workspace

import (
	"context"
	"fmt"
	"log/slog"
)

// RemoveOptions contains options for removing a workspace
type RemoveOptions struct {
	Force bool // Force removal even if workspace is in use
}

// SessionStopper is an interface for stopping sessions
type SessionStopper interface {
	StopSessionsInWorkspace(ctx context.Context, workspaceID string) error
	ListSessionsInWorkspace(ctx context.Context, workspaceID string) ([]string, error)
}

// SemaphoreComponents holds semaphore-related components
type SemaphoreComponents struct {
	SemaphoreManager *SemaphoreManager
	Reconciler       *SemaphoreReconciler
	SessionStopper   SessionStopper
}

// InitializeSemaphore sets up semaphore components for the workspace manager
func (m *Manager) InitializeSemaphore(sessionChecker SessionChecker, sessionStopper SessionStopper, logger *slog.Logger) {
	// Create semaphore manager
	semaphoreManager := NewSemaphoreManager(m.workspacesDir, sessionChecker)

	// Create reconciler
	reconciler := NewSemaphoreReconciler(semaphoreManager, sessionChecker, logger)
	semaphoreManager.SetReconciler(reconciler)

	// Store components
	m.semaphore = &SemaphoreComponents{
		SemaphoreManager: semaphoreManager,
		Reconciler:       reconciler,
		SessionStopper:   sessionStopper,
	}

	// Debug log
	if logger != nil {
		logger.Debug("Semaphore initialized", "workspacesDir", m.workspacesDir)
	}
}

// AcquireSemaphore acquires a semaphore for a workspace
func (m *Manager) AcquireSemaphore(workspaceID string, holder Holder) error {
	if m.semaphore == nil || m.semaphore.SemaphoreManager == nil {
		// Semaphore not initialized, skip
		// Semaphore not initialized, skip silently
		return nil
	}

	// Proceed with acquisition
	return m.semaphore.SemaphoreManager.Acquire(workspaceID, holder)
}

// ReleaseSemaphore releases a semaphore for a workspace
func (m *Manager) ReleaseSemaphore(workspaceID string, holderID string) error {
	if m.semaphore == nil || m.semaphore.SemaphoreManager == nil {
		// Semaphore not initialized, skip
		return nil
	}

	return m.semaphore.SemaphoreManager.Release(workspaceID, holderID)
}

// GetSemaphoreHolders returns all holders for a workspace
func (m *Manager) GetSemaphoreHolders(workspaceID string) ([]Holder, error) {
	if m.semaphore == nil || m.semaphore.SemaphoreManager == nil {
		// Semaphore not initialized, return empty
		return []Holder{}, nil
	}

	return m.semaphore.SemaphoreManager.GetHolders(workspaceID)
}

// populateSemaphoreHolders populates the SemaphoreHolders field in a workspace
func (m *Manager) populateSemaphoreHolders(workspace *Workspace) {
	if m.semaphore == nil || m.semaphore.SemaphoreManager == nil {
		// Semaphore not initialized, leave empty
		workspace.SemaphoreHolders = []Holder{}
		return
	}

	holders, err := m.semaphore.SemaphoreManager.GetHolders(workspace.ID)
	if err != nil {
		// On error, leave empty
		workspace.SemaphoreHolders = []Holder{}
		return
	}

	workspace.SemaphoreHolders = holders
}

// IsWorkspaceInUse checks if a workspace is in use by any holders
func (m *Manager) IsWorkspaceInUse(workspaceID string) (bool, []Holder, error) {
	if m.semaphore == nil || m.semaphore.SemaphoreManager == nil {
		// Semaphore not initialized, assume not in use
		return false, []Holder{}, nil
	}

	// Reconcile before checking
	if m.semaphore.Reconciler != nil {
		_ = m.semaphore.Reconciler.ReconcileWorkspace(workspaceID)
	}

	return m.semaphore.SemaphoreManager.IsInUse(workspaceID)
}

// RemoveWithOptions removes a workspace with additional options
func (m *Manager) RemoveWithOptions(ctx context.Context, identifier Identifier, opts RemoveOptions) error {
	workspace, err := m.ResolveWorkspace(ctx, identifier)
	if err != nil {
		return err
	}

	// Check semaphore if not forcing
	if !opts.Force && m.semaphore != nil {
		inUse, holders, err := m.IsWorkspaceInUse(workspace.ID)
		if err != nil {
			return fmt.Errorf("failed to check workspace usage: %w", err)
		}

		if inUse {
			return &ErrWorkspaceInUse{
				WorkspaceID: workspace.ID,
				Holders:     holders,
			}
		}
	}

	// If forcing, stop all sessions first
	if opts.Force && m.semaphore != nil && m.semaphore.SessionStopper != nil {
		if err := m.semaphore.SessionStopper.StopSessionsInWorkspace(ctx, workspace.ID); err != nil {
			return fmt.Errorf("failed to stop sessions: %w", err)
		}
	}

	// Call the original Remove method
	return m.Remove(ctx, identifier)
}

// ErrWorkspaceInUse is returned when trying to remove a workspace that is in use
type ErrWorkspaceInUse struct {
	WorkspaceID string
	Holders     []Holder
}

func (e *ErrWorkspaceInUse) Error() string {
	return fmt.Sprintf("workspace %s is currently in use by %d holder(s)", e.WorkspaceID, len(e.Holders))
}

// GetHolderDetails returns formatted details about holders
func (e *ErrWorkspaceInUse) GetHolderDetails() []string {
	details := make([]string, len(e.Holders))
	for i, h := range e.Holders {
		desc := h.Description
		if desc == "" {
			desc = string(h.Type)
		}
		details[i] = fmt.Sprintf("%s (%s)", h.ID, desc)
	}
	return details
}
