package session_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
	runtimeinit "github.com/aki/amux/internal/runtime/init"
	"github.com/aki/amux/internal/tests/helpers"
)

func TestSessionSemaphoreIntegration(t *testing.T) {
	// Initialize runtime registry for tests
	if err := runtimeinit.RegisterDefaults(); err != nil {
		t.Fatalf("Failed to initialize runtimes: %v", err)
	}

	// Check if tmux is available for session tests
	adapter, err := tmux.NewAdapter()
	if err != nil || adapter == nil || !adapter.IsAvailable() {
		t.Skip("Skipping test: tmux not available")
	}

	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()

	// Add a default agent for testing
	cfg.Agents = map[string]config.Agent{
		"default": {
			Name:        "Default Agent",
			Runtime:     "tmux",
			Description: "Default test agent",
			Command:     []string{"echo", "test"},
		},
	}

	err = configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	// Create session ID mapper
	sessionIDMapper, err := idmap.NewSessionIDMapper(configManager.GetAmuxDir())
	require.NoError(t, err)

	// Create session manager
	sessManager, err := session.NewManager(
		configManager.GetAmuxDir(),
		wsManager,
		configManager,
		sessionIDMapper,
	)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("SessionAcquiresWorkspaceSemaphore", func(t *testing.T) {
		// Create workspace
		ws, err := wsManager.Create(ctx, workspace.CreateOptions{
			Name:        "test-semaphore-ws",
			Description: "Test workspace for semaphore",
		})
		require.NoError(t, err)

		// Create session (without starting)
		sess, err := sessManager.CreateSession(ctx, session.Options{
			WorkspaceID: ws.ID,
			AgentID:     "default",
			Type:        session.TypeTmux,
		})
		require.NoError(t, err)

		// Check no sessions hold the workspace initially
		sessionIDs, err := ws.SessionIDs()
		assert.NoError(t, err)
		assert.Empty(t, sessionIDs)

		// Since we can't easily mock tmux operations in this test,
		// we'll test the semaphore functionality directly through workspace
		// The actual integration is tested in the manager tests

		// Simulate what happens when a session starts
		err = ws.Acquire(sess.ID())
		assert.NoError(t, err)

		// Check session holds the workspace
		sessionIDs, err = ws.SessionIDs()
		assert.NoError(t, err)
		assert.Contains(t, sessionIDs, sess.ID())

		// Simulate what happens when a session stops
		err = ws.Release(sess.ID())
		assert.NoError(t, err)

		// Check session no longer holds the workspace
		sessionIDs, err = ws.SessionIDs()
		assert.NoError(t, err)
		assert.NotContains(t, sessionIDs, sess.ID())
	})

	t.Run("ReconcileStaleWorkspaceSemaphores", func(t *testing.T) {
		// Create workspace
		ws, err := wsManager.Create(ctx, workspace.CreateOptions{
			Name:        "test-reconcile-ws",
			Description: "Test workspace for reconciliation",
		})
		require.NoError(t, err)

		// Manually acquire semaphore with fake session ID
		fakeSessionID := "fake-session-123"
		err = ws.Acquire(fakeSessionID)
		assert.NoError(t, err)

		// Check fake session holds the workspace
		sessionIDs, err := ws.SessionIDs()
		assert.NoError(t, err)
		assert.Contains(t, sessionIDs, fakeSessionID)

		// Run reconciliation
		err = sessManager.ReconcileWorkspaceSemaphores(ctx)
		assert.NoError(t, err)

		// Check fake session no longer holds the workspace
		sessionIDs, err = ws.SessionIDs()
		assert.NoError(t, err)
		assert.NotContains(t, sessionIDs, fakeSessionID)
	})

	t.Run("MultipleSessionsCanShareWorkspace", func(t *testing.T) {
		// Create workspace
		ws, err := wsManager.Create(ctx, workspace.CreateOptions{
			Name:        "test-shared-ws",
			Description: "Test workspace for multiple sessions",
		})
		require.NoError(t, err)

		// Create two sessions
		sess1, err := sessManager.CreateSession(ctx, session.Options{
			WorkspaceID: ws.ID,
			AgentID:     "default",
			Type:        session.TypeTmux,
		})
		require.NoError(t, err)

		sess2, err := sessManager.CreateSession(ctx, session.Options{
			WorkspaceID: ws.ID,
			AgentID:     "default",
			Type:        session.TypeTmux,
		})
		require.NoError(t, err)

		// Simulate both sessions acquiring the workspace
		err = ws.Acquire(sess1.ID())
		assert.NoError(t, err)
		err = ws.Acquire(sess2.ID())
		assert.NoError(t, err)

		// Check both sessions hold the workspace
		sessionIDs, err := ws.SessionIDs()
		assert.NoError(t, err)
		assert.Len(t, sessionIDs, 2)
		assert.Contains(t, sessionIDs, sess1.ID())
		assert.Contains(t, sessionIDs, sess2.ID())

		// Check workspace session count
		count := ws.SessionCount()
		assert.Equal(t, 2, count)
	})
}
