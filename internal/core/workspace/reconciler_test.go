package workspace

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSemaphoreReconciler_ReconcileWorkspace(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "workspaces")
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	checker := &mockSessionChecker{
		activeSessions: map[string]bool{
			"session-1": true,
			"session-2": false, // Inactive
			// session-3 doesn't exist
		},
	}

	sm := NewSemaphoreManager(basePath, checker)
	reconciler := NewSemaphoreReconciler(sm, checker, logger)
	// Don't set reconciler yet to avoid auto-reconciliation during setup

	workspaceID := "ws-reconcile-test"

	// Add various holders
	holders := []Holder{
		{ID: "session-1", Type: HolderTypeSession, SessionID: "session-1", Description: "Active session"},
		{ID: "session-2", Type: HolderTypeSession, SessionID: "session-2", Description: "Inactive session"},
		{ID: "session-3", Type: HolderTypeSession, SessionID: "session-3", Description: "Non-existent session"},
		{ID: "cli-1", Type: HolderTypeCLI, Timestamp: time.Now().Add(-2 * time.Minute), Description: "Recent CLI"},
		{ID: "cli-2", Type: HolderTypeCLI, Timestamp: time.Now().Add(-10 * time.Minute), Description: "Expired CLI"},
		{ID: "unknown-1", Type: "unknown", Description: "Unknown type"},
	}

	for _, h := range holders {
		// Set timestamp for holders that need it
		if h.Timestamp.IsZero() && h.Type != HolderTypeCLI {
			h.Timestamp = time.Now()
		}
		err := sm.Acquire(workspaceID, h)
		require.NoError(t, err)
	}

	// Verify all holders were added
	beforeHolders, err := sm.GetHolders(workspaceID)
	require.NoError(t, err)
	assert.Len(t, beforeHolders, 6)

	// Reconcile
	err = reconciler.ReconcileWorkspace(workspaceID)
	require.NoError(t, err)

	// Verify only valid holders remain
	afterHolders, err := sm.GetHolders(workspaceID)
	require.NoError(t, err)
	assert.Len(t, afterHolders, 2) // Only session-1 and cli-1 should remain

	// Check specific holders
	holderIDs := make(map[string]bool)
	for _, h := range afterHolders {
		holderIDs[h.ID] = true
	}
	assert.True(t, holderIDs["session-1"], "Active session should remain")
	assert.True(t, holderIDs["cli-1"], "Recent CLI should remain")
	assert.False(t, holderIDs["session-2"], "Inactive session should be removed")
	assert.False(t, holderIDs["session-3"], "Non-existent session should be removed")
	assert.False(t, holderIDs["cli-2"], "Expired CLI should be removed")
	assert.False(t, holderIDs["unknown-1"], "Unknown type should be removed")
}

func TestSemaphoreReconciler_ReconcileAll(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "workspaces")
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	checker := &mockSessionChecker{
		activeSessions: map[string]bool{
			"session-1": true,
		},
	}

	sm := NewSemaphoreManager(basePath, checker)
	reconciler := NewSemaphoreReconciler(sm, checker, logger)

	// Create multiple workspaces with holders
	workspaceIDs := []string{"ws-1", "ws-2", "ws-3"}

	for _, wsID := range workspaceIDs {
		// Add a mix of valid and invalid holders
		holders := []Holder{
			{ID: "session-1", Type: HolderTypeSession, SessionID: "session-1"},
			{ID: "session-dead", Type: HolderTypeSession, SessionID: "session-dead"},
			{ID: "cli-old", Type: HolderTypeCLI, Timestamp: time.Now().Add(-1 * time.Hour)},
		}

		for _, h := range holders {
			if h.Timestamp.IsZero() && h.Type != HolderTypeCLI {
				h.Timestamp = time.Now()
			}
			err := sm.Acquire(wsID, h)
			require.NoError(t, err)
		}
	}

	// Reconcile all
	err := reconciler.ReconcileAll(workspaceIDs)
	require.NoError(t, err)

	// Verify each workspace has only valid holders
	for _, wsID := range workspaceIDs {
		holders, err := sm.GetHolders(wsID)
		require.NoError(t, err)
		assert.Len(t, holders, 1, "Each workspace should have only 1 valid holder")
		assert.Equal(t, "session-1", holders[0].ID)
	}
}

func TestSemaphoreReconciler_ReconcileOnAcquire(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "workspaces")
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	checker := &mockSessionChecker{
		activeSessions: map[string]bool{
			"session-new": true,
		},
	}

	sm := NewSemaphoreManager(basePath, checker)
	reconciler := NewSemaphoreReconciler(sm, checker, logger)
	sm.SetReconciler(reconciler)

	workspaceID := "ws-acquire-test"

	// Add an expired holder directly (bypassing reconciliation)
	sm.updateSemaphore(workspaceID, func(data *SemaphoreData) error {
		data.Holders = append(data.Holders, Holder{
			ID:        "cli-old",
			Type:      HolderTypeCLI,
			Timestamp: time.Now().Add(-1 * time.Hour),
		})
		return nil
	})

	// Verify expired holder exists
	holders, err := sm.GetHolders(workspaceID)
	require.NoError(t, err)
	assert.Len(t, holders, 1)

	// Acquire new holder (should trigger reconciliation)
	newHolder := Holder{
		ID:        "session-new",
		Type:      HolderTypeSession,
		SessionID: "session-new",
	}
	err = sm.Acquire(workspaceID, newHolder)
	require.NoError(t, err)

	// Verify old holder was removed and new holder added
	holders, err = sm.GetHolders(workspaceID)
	require.NoError(t, err)
	assert.Len(t, holders, 1)
	assert.Equal(t, "session-new", holders[0].ID)
}

func TestReconciler_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	basePath := filepath.Join(tempDir, "workspaces")
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create a reconciler with nil session checker to cause errors
	sm := NewSemaphoreManager(basePath, nil)
	reconciler := NewSemaphoreReconciler(sm, nil, logger)

	// Try to reconcile non-existent workspace
	err := reconciler.ReconcileWorkspace("non-existent")
	// Should not error on non-existent workspace
	assert.NoError(t, err)

	// ReconcileAll with some failing workspaces
	workspaceIDs := []string{"ws-1", "ws-2", "ws-3"}

	// Create one workspace with a problematic path
	problemPath := filepath.Join(basePath, "ws-2")
	err = os.MkdirAll(problemPath, 0o755)
	require.NoError(t, err)

	// Make the semaphore file unreadable
	semaphorePath := filepath.Join(problemPath, semaphoreFileName)
	err = os.WriteFile(semaphorePath, []byte("data"), 0o000)
	require.NoError(t, err)

	// Reconcile all - should report errors but not panic
	err = reconciler.ReconcileAll(workspaceIDs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reconciliation completed with")
}
