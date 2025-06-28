package workspace_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/tests/helpers"
)

func TestManager_WorkspaceSemaphore(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	manager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a test workspace
	opts := workspace.CreateOptions{
		Name:        "test-semaphore",
		Description: "Test workspace for semaphore",
	}
	ws, err := manager.Create(ctx, opts)
	require.NoError(t, err)

	t.Run("AcquireAndRelease", func(t *testing.T) {
		sessionID := "session-123"

		// Acquire workspace
		err := ws.Acquire(sessionID)
		assert.NoError(t, err)

		// Check session is listed
		sessions, err := ws.SessionIDs()
		assert.NoError(t, err)
		assert.Contains(t, sessions, sessionID)

		// Release workspace
		err = ws.Release(sessionID)
		assert.NoError(t, err)

		// Check session is no longer listed
		sessions, err = ws.SessionIDs()
		assert.NoError(t, err)
		assert.NotContains(t, sessions, sessionID)
	})

	t.Run("MultipleSessionsCanAcquire", func(t *testing.T) {
		session1 := "session-001"
		session2 := "session-002"

		// Both sessions acquire
		err := ws.Acquire(session1)
		assert.NoError(t, err)
		err = ws.Acquire(session2)
		assert.NoError(t, err)

		// Check both are listed
		sessions, err := ws.SessionIDs()
		assert.NoError(t, err)
		assert.Len(t, sessions, 2)
		assert.Contains(t, sessions, session1)
		assert.Contains(t, sessions, session2)

		// Release both
		err = ws.Release(session1)
		assert.NoError(t, err)
		err = ws.Release(session2)
		assert.NoError(t, err)
	})

	t.Run("SameSessionCannotAcquireTwice", func(t *testing.T) {
		sessionID := "session-456"

		// First acquire should succeed
		err := ws.Acquire(sessionID)
		assert.NoError(t, err)

		// Second acquire should fail
		err = ws.Acquire(sessionID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already held")

		// Clean up
		err = ws.Release(sessionID)
		assert.NoError(t, err)
	})

	t.Run("IsWorkspaceAvailable", func(t *testing.T) {
		// Should be available initially
		available, err := ws.IsAvailable()
		assert.NoError(t, err)
		assert.True(t, available)

		// Acquire by one session - should still be available
		sessionID := "session-789"
		err = ws.Acquire(sessionID)
		assert.NoError(t, err)

		available, err = ws.IsAvailable()
		assert.NoError(t, err)
		assert.True(t, available) // Still has capacity

		// Clean up
		err = ws.Release(sessionID)
		assert.NoError(t, err)
	})

	t.Run("RemoveWithSessionCheck", func(t *testing.T) {
		// Create another workspace for removal test
		opts2 := workspace.CreateOptions{
			Name:        "test-remove-check",
			Description: "Test workspace for removal check",
		}
		ws2, err := manager.Create(ctx, opts2)
		require.NoError(t, err)

		// Acquire by a session
		sessionID := "session-remove"
		err = ws2.Acquire(sessionID)
		assert.NoError(t, err)

		// Try to remove without force - should fail
		err = manager.RemoveWithSessionCheck(ctx, workspace.Identifier(ws2.ID), false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "currently in use")

		// Remove with force - should succeed
		err = manager.RemoveWithSessionCheck(ctx, workspace.Identifier(ws2.ID), true)
		assert.NoError(t, err)

		// Verify workspace is removed
		_, err = manager.Get(ctx, workspace.ID(ws2.ID))
		assert.Error(t, err)
	})

	t.Run("WorkspaceWithNoSemaphore", func(t *testing.T) {
		// Create workspace without any semaphore operations
		opts3 := workspace.CreateOptions{
			Name:        "test-no-semaphore",
			Description: "Test workspace without semaphore",
		}
		ws3, err := manager.Create(ctx, opts3)
		require.NoError(t, err)

		// Check sessions - should return empty list
		sessions, err := ws3.SessionIDs()
		assert.NoError(t, err)
		assert.Empty(t, sessions)

		// Check availability - should be available
		available, err := ws3.IsAvailable()
		assert.NoError(t, err)
		assert.True(t, available)

		// Can remove without force
		err = manager.RemoveWithSessionCheck(ctx, workspace.Identifier(ws3.ID), false)
		assert.NoError(t, err)
	})
}

func TestManager_WorkspaceSemaphoreEdgeCases(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	manager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("NonExistentWorkspace", func(t *testing.T) {
		// Try RemoveWithSessionCheck on non-existent workspace
		fakeID := "workspace-fake-123"

		err := manager.RemoveWithSessionCheck(ctx, workspace.Identifier(fakeID), false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace not found")
	})

	t.Run("ReleaseNonHeldSession", func(t *testing.T) {
		// Create workspace
		opts := workspace.CreateOptions{
			Name:        "test-release-nonheld",
			Description: "Test releasing non-held session",
		}
		ws, err := manager.Create(ctx, opts)
		require.NoError(t, err)

		// Try to release a session that never acquired
		err = ws.Release("non-existent-session")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not held")
	})
}
