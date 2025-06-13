package session

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/tests/helpers"
)

func TestRemoveSession(t *testing.T) {
	// Skip if tmux is not available
	tmuxAdapter, err := tmux.NewAdapter()
	if err != nil || !tmuxAdapter.IsAvailable() {
		t.Skip("tmux not available on this system")
	}

	// Create test repository
	repoDir := helpers.CreateTestRepo(t)
	defer os.RemoveAll(repoDir)

	// Change to test directory to ensure commands work correctly
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(repoDir)
	require.NoError(t, err)
	defer os.Chdir(oldWd)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	err = configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	// Create a workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name: "test-workspace",
	})
	require.NoError(t, err)

	// Create session manager
	sessionManager, err := createSessionManager(configManager, wsManager)
	require.NoError(t, err)

	t.Run("cannot remove running session", func(t *testing.T) {
		// Create a test session
		sess, err := sessionManager.CreateSession(session.Options{
			WorkspaceID: ws.ID,
			AgentID:     "claude",
		})
		require.NoError(t, err)

		// Start the session
		ctx := context.TODO()
		err = sess.Start(ctx)
		require.NoError(t, err)

		// Try to remove it
		cmd := removeCmd()
		cmd.SetArgs([]string{sess.ID()})
		err = cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot remove running session")

		// Clean up: stop the session
		err = sess.Stop()
		require.NoError(t, err)
	})

	t.Run("can remove stopped session", func(t *testing.T) {
		// Create a new test session
		sess, err := sessionManager.CreateSession(session.Options{
			WorkspaceID: ws.ID,
			AgentID:     "claude",
		})
		require.NoError(t, err)

		// Start and then stop the session
		ctx := context.TODO()
		err = sess.Start(ctx)
		require.NoError(t, err)
		err = sess.Stop()
		require.NoError(t, err)

		// Now remove it
		cmd := removeCmd()
		cmd.SetArgs([]string{sess.ID()})
		err = cmd.Execute()
		assert.NoError(t, err)

		// Verify it's gone by trying to remove again
		cmd2 := removeCmd()
		cmd2.SetArgs([]string{sess.ID()})
		err = cmd2.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get session")
	})

	t.Run("error on non-existent session", func(t *testing.T) {
		cmd := removeCmd()
		cmd.SetArgs([]string{"non-existent-session"})
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get session")
	})
}
