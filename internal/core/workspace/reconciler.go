package workspace

import (
	"fmt"
	"log/slog"
)

// SemaphoreReconciler handles cleanup of stale semaphore entries
type SemaphoreReconciler struct {
	semaphoreManager *SemaphoreManager
	sessionChecker   SessionChecker
	logger           *slog.Logger
}

// NewSemaphoreReconciler creates a new semaphore reconciler
func NewSemaphoreReconciler(semaphoreManager *SemaphoreManager, sessionChecker SessionChecker, logger *slog.Logger) *SemaphoreReconciler {
	return &SemaphoreReconciler{
		semaphoreManager: semaphoreManager,
		sessionChecker:   sessionChecker,
		logger:           logger,
	}
}

// ReconcileWorkspace cleans up stale holders for a specific workspace
func (r *SemaphoreReconciler) ReconcileWorkspace(workspaceID string) error {
	return r.semaphoreManager.updateSemaphore(workspaceID, func(data *SemaphoreData) error {
		validHolders := make([]Holder, 0, len(data.Holders))
		removedCount := 0

		for _, holder := range data.Holders {
			if r.isHolderValid(holder) {
				validHolders = append(validHolders, holder)
			} else {
				removedCount++
				r.logger.Debug("removing stale semaphore holder",
					"workspace_id", workspaceID,
					"holder_id", holder.ID,
					"holder_type", holder.Type,
					"session_id", holder.SessionID,
				)
			}
		}

		data.Holders = validHolders

		if removedCount > 0 {
			r.logger.Debug("reconciled workspace semaphore",
				"workspace_id", workspaceID,
				"removed_count", removedCount,
				"remaining_count", len(validHolders),
			)
		}

		return nil
	})
}

// ReconcileAll reconciles semaphores for all workspaces
func (r *SemaphoreReconciler) ReconcileAll(workspaceIDs []string) error {
	var reconcileErrors []error
	successCount := 0

	for _, wsID := range workspaceIDs {
		if err := r.ReconcileWorkspace(wsID); err != nil {
			reconcileErrors = append(reconcileErrors, fmt.Errorf("workspace %s: %w", wsID, err))
			r.logger.Error("failed to reconcile workspace semaphore",
				"workspace_id", wsID,
				"error", err,
			)
		} else {
			successCount++
		}
	}

	if len(reconcileErrors) > 0 {
		return fmt.Errorf("reconciliation completed with %d errors (succeeded: %d)", len(reconcileErrors), successCount)
	}

	return nil
}

// isHolderValid checks if a holder is still valid
func (r *SemaphoreReconciler) isHolderValid(holder Holder) bool {
	// Check if holder has expired based on type
	if holder.IsExpired() {
		return false
	}

	switch holder.Type {
	case HolderTypeSession:
		// Check if session exists and is active
		active, err := r.sessionChecker.IsSessionActive(holder.SessionID)
		if err != nil {
			// If we can't check, assume it's invalid to be safe
			r.logger.Warn("failed to check session status",
				"session_id", holder.SessionID,
				"error", err,
			)
			return false
		}
		return active

	case HolderTypeCLI:
		// CLI holders are valid if not expired (already checked above)
		return true

	default:
		// Unknown holder types are invalid
		r.logger.Warn("unknown holder type",
			"holder_type", holder.Type,
			"holder_id", holder.ID,
		)
		return false
	}
}

// ReconcileOnAcquire reconciles before acquiring a new semaphore
func (r *SemaphoreReconciler) ReconcileOnAcquire(workspaceID string) error {
	// Perform reconciliation but don't fail the acquire if reconciliation fails
	if err := r.ReconcileWorkspace(workspaceID); err != nil {
		r.logger.Warn("reconciliation failed during acquire",
			"workspace_id", workspaceID,
			"error", err,
		)
		// Continue with acquire even if reconciliation fails
	}
	return nil
}
