package workspace_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/config"
	"github.com/aki/amux/internal/tests/helpers"
	"github.com/aki/amux/internal/workspace"
)

func TestWorkspaceSessionProtection(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("CannotRemoveWorkspaceWithActiveSessions", func(t *testing.T) {
		// Create a test workspace
		ws, err := wsManager.Create(ctx, workspace.CreateOptions{
			Name:        "test-remove-protected",
			Description: "Test workspace for protection",
		})
		require.NoError(t, err)

		// Simulate an active session by acquiring the semaphore
		fakeSessionID := "test-session-123"
		err = ws.Acquire(fakeSessionID)
		require.NoError(t, err)

		// Try to remove without force using RemoveWithSessionCheck
		err = wsManager.RemoveWithSessionCheck(ctx, workspace.Identifier(ws.ID), false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "currently in use")

		// Verify workspace still exists
		exists, err := wsManager.Get(ctx, workspace.ID(ws.ID))
		assert.NoError(t, err)
		assert.NotNil(t, exists)

		// Clean up
		err = ws.Release(fakeSessionID)
		assert.NoError(t, err)
		err = wsManager.Remove(ctx, workspace.Identifier(ws.ID), workspace.RemoveOptions{})
		assert.NoError(t, err)
	})

	t.Run("CanForceRemoveWorkspaceWithActiveSessions", func(t *testing.T) {
		// Create a test workspace
		ws, err := wsManager.Create(ctx, workspace.CreateOptions{
			Name:        "test-force-remove",
			Description: "Test workspace for force removal",
		})
		require.NoError(t, err)

		// Simulate an active session by acquiring the semaphore
		fakeSessionID := "test-session-456"
		err = ws.Acquire(fakeSessionID)
		require.NoError(t, err)

		// Force remove should succeed
		err = wsManager.RemoveWithSessionCheck(ctx, workspace.Identifier(ws.ID), true)
		assert.NoError(t, err)

		// Verify workspace no longer exists
		_, err = wsManager.Get(ctx, workspace.ID(ws.ID))
		assert.Error(t, err)
	})
}
