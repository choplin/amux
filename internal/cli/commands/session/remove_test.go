package session

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
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

	// Create a workspace with unique name
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name: fmt.Sprintf("test-workspace-%s", uuid.New().String()[:8]),
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
		if err != nil {
			assert.Contains(t, err.Error(), "failed to get session")
		}
	})

	t.Run("error on non-existent session", func(t *testing.T) {
		cmd := removeCmd()
		cmd.SetArgs([]string{"non-existent-session"})
		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get session")
	})

	t.Run("auto-removes workspace when session is removed", func(t *testing.T) {
		// Create a new workspace marked as auto-created with unique name
		wsAuto, err := wsManager.Create(workspace.CreateOptions{
			Name:        fmt.Sprintf("test-workspace-auto-%s", uuid.New().String()[:8]),
			AutoCreated: true,
		})
		require.NoError(t, err)

		// Create a session in the auto-created workspace
		sess, err := sessionManager.CreateSession(session.Options{
			WorkspaceID: wsAuto.ID,
			AgentID:     "claude",
		})
		require.NoError(t, err)

		// Start and then stop the session
		ctx := context.TODO()
		err = sess.Start(ctx)
		require.NoError(t, err)
		err = sess.Stop()
		require.NoError(t, err)

		// Verify workspace exists before removal
		_, err = wsManager.ResolveWorkspace(workspace.Identifier(wsAuto.ID))
		require.NoError(t, err)

		// Remove the session
		cmd := removeCmd()
		cmd.SetArgs([]string{sess.ID()})
		err = cmd.Execute()
		assert.NoError(t, err)

		// Verify workspace was removed
		_, err = wsManager.ResolveWorkspace(workspace.Identifier(wsAuto.ID))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace not found")
	})

	t.Run("keeps workspace when --keep-workspace flag is used", func(t *testing.T) {
		// Create a new workspace marked as auto-created with unique name
		ws2, err := wsManager.Create(workspace.CreateOptions{
			Name:        fmt.Sprintf("test-workspace-keep-%s", uuid.New().String()[:8]),
			AutoCreated: true,
		})
		require.NoError(t, err)

		// Create a session in the auto-created workspace
		sess, err := sessionManager.CreateSession(session.Options{
			WorkspaceID: ws2.ID,
			AgentID:     "claude",
		})
		require.NoError(t, err)

		// Start and then stop the session
		ctx := context.TODO()
		err = sess.Start(ctx)
		require.NoError(t, err)
		err = sess.Stop()
		require.NoError(t, err)

		// Remove the session with --keep-workspace flag
		cmd := removeCmd()
		cmd.SetArgs([]string{sess.ID(), "--keep-workspace"})
		err = cmd.Execute()
		assert.NoError(t, err)

		// Verify workspace still exists
		_, err = wsManager.ResolveWorkspace(workspace.Identifier(ws2.ID))
		assert.NoError(t, err)
	})

	t.Run("does not remove workspace when used by other sessions", func(t *testing.T) {
		// Create a new workspace marked as auto-created with unique name
		wsName := fmt.Sprintf("test-workspace-multi-%s", uuid.New().String()[:8])
		ws3, err := wsManager.Create(workspace.CreateOptions{
			Name:        wsName,
			AutoCreated: true,
		})
		require.NoError(t, err)

		// Ensure workspace cleanup happens after all operations
		t.Cleanup(func() {
			// Try to remove workspace if it still exists
			if _, err := wsManager.ResolveWorkspace(workspace.Identifier(ws3.ID)); err == nil {
				_ = wsManager.Remove(workspace.Identifier(ws3.ID))
			}
		})

		// Create first session in the auto-created workspace
		sess1, err := sessionManager.CreateSession(session.Options{
			WorkspaceID: ws3.ID,
			AgentID:     "claude",
		})
		require.NoError(t, err)

		// Create second session in the same workspace
		sess2, err := sessionManager.CreateSession(session.Options{
			WorkspaceID: ws3.ID,
			AgentID:     "claude",
		})
		require.NoError(t, err)

		// Start and stop both sessions
		ctx := context.TODO()
		err = sess1.Start(ctx)
		require.NoError(t, err)
		err = sess1.Stop()
		require.NoError(t, err)

		err = sess2.Start(ctx)
		require.NoError(t, err)
		err = sess2.Stop()
		require.NoError(t, err)

		// Store session IDs
		sess1ID := sess1.ID()
		sess2ID := sess2.ID()

		// Verify both sessions exist before removal
		sessions, err := sessionManager.ListSessions()
		require.NoError(t, err)
		sessionCount := 0
		for _, s := range sessions {
			if s.Info().WorkspaceID == ws3.ID {
				sessionCount++
			}
		}
		assert.Equal(t, 2, sessionCount, "Expected 2 sessions in workspace before removal")

		// Remove the first session
		cmd := removeCmd()
		cmd.SetArgs([]string{sess1ID})
		err = cmd.Execute()
		assert.NoError(t, err)

		// Verify workspace still exists (because sess2 is using it)
		_, err = wsManager.ResolveWorkspace(workspace.Identifier(ws3.ID))
		assert.NoError(t, err, "Workspace should still exist after removing first session")

		// Verify only one session remains
		sessions, err = sessionManager.ListSessions()
		require.NoError(t, err)
		sessionCount = 0
		for _, s := range sessions {
			if s.Info().WorkspaceID == ws3.ID {
				sessionCount++
			}
		}
		assert.Equal(t, 1, sessionCount, "Expected 1 session in workspace after first removal")

		// Remove the second session
		cmd2 := removeCmd()
		cmd2.SetArgs([]string{sess2ID})
		err = cmd2.Execute()
		assert.NoError(t, err)

		// Workspace should be removed now (it was auto-created and no sessions are using it)
		_, err = wsManager.ResolveWorkspace(workspace.Identifier(ws3.ID))
		assert.Error(t, err, "Workspace should be removed after removing last session")
		if err != nil {
			assert.Contains(t, err.Error(), "workspace not found")
		}
	})

	t.Run("does not remove manually created workspace", func(t *testing.T) {
		// Create a new workspace WITHOUT auto-created flag with unique name
		ws4, err := wsManager.Create(workspace.CreateOptions{
			Name:        fmt.Sprintf("test-workspace-manual-%s", uuid.New().String()[:8]),
			AutoCreated: false, // explicitly not auto-created
		})
		require.NoError(t, err)

		// Create a session in the manually created workspace
		sess, err := sessionManager.CreateSession(session.Options{
			WorkspaceID: ws4.ID,
			AgentID:     "claude",
		})
		require.NoError(t, err)

		// Start and then stop the session
		ctx := context.TODO()
		err = sess.Start(ctx)
		require.NoError(t, err)
		err = sess.Stop()
		require.NoError(t, err)

		// Remove the session
		cmd := removeCmd()
		cmd.SetArgs([]string{sess.ID()})
		err = cmd.Execute()
		assert.NoError(t, err)

		// Verify workspace still exists
		_, err = wsManager.ResolveWorkspace(workspace.Identifier(ws4.ID))
		assert.NoError(t, err)
	})
}
