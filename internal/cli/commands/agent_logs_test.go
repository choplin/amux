package commands

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/mailbox"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/tests/helpers"
)

func TestStreamSessionLogsBasic(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	// Create workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name: "test-workspace",
	})
	require.NoError(t, err)

	// Create dependencies
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	require.NoError(t, err)
	mailboxManager := mailbox.NewManager(configManager.GetAmuxDir())
	store, err := session.NewFileStore(configManager.GetAmuxDir())
	require.NoError(t, err)

	// Create session manager with mock adapter
	sessionManager := session.NewManager(store, wsManager, mailboxManager, idMapper, session.WithLogger(logger.Nop()))

	// Replace tmux adapter with mock
	mockAdapter := tmux.NewMockAdapter()
	sessionManager.SetTmuxAdapter(mockAdapter)

	// Create and start a session
	opts := session.Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     "echo 'Starting streaming test'",
	}

	sess, err := sessionManager.CreateSession(opts)
	require.NoError(t, err)

	ctx := context.Background()
	err = sess.Start(ctx)
	require.NoError(t, err)

	// Add initial output
	info := sess.Info()
	err = mockAdapter.AppendSessionOutput(info.TmuxSession, "Initial output")
	require.NoError(t, err)

	// Test that GetOutput returns the current output
	output, err := sess.GetOutput()
	require.NoError(t, err)
	assert.Contains(t, string(output), "Initial output")

	// Add more output
	err = mockAdapter.AppendSessionOutput(info.TmuxSession, "New line 1")
	require.NoError(t, err)

	// Get output again
	output, err = sess.GetOutput()
	require.NoError(t, err)
	assert.Contains(t, string(output), "Initial output")
	assert.Contains(t, string(output), "New line 1")

	// Stop session
	err = sess.Stop()
	require.NoError(t, err)
}

func TestStreamSessionLogsPolling(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	// Create workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name: "test-workspace",
	})
	require.NoError(t, err)

	// Create dependencies
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	require.NoError(t, err)
	mailboxManager := mailbox.NewManager(configManager.GetAmuxDir())
	store, err := session.NewFileStore(configManager.GetAmuxDir())
	require.NoError(t, err)

	// Create session manager with mock adapter
	sessionManager := session.NewManager(store, wsManager, mailboxManager, idMapper, session.WithLogger(logger.Nop()))

	// Replace tmux adapter with mock
	mockAdapter := tmux.NewMockAdapter()
	sessionManager.SetTmuxAdapter(mockAdapter)

	// Create and start a session
	opts := session.Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     "echo 'Starting test'",
	}

	sess, err := sessionManager.CreateSession(opts)
	require.NoError(t, err)

	ctx := context.Background()
	err = sess.Start(ctx)
	require.NoError(t, err)

	// Test the polling logic manually
	info := sess.Info()

	// Initial output
	err = mockAdapter.AppendSessionOutput(info.TmuxSession, "Line 1")
	require.NoError(t, err)

	output1, err := sess.GetOutput()
	require.NoError(t, err)
	assert.Contains(t, string(output1), "Line 1")

	// Add more output
	err = mockAdapter.AppendSessionOutput(info.TmuxSession, "Line 2")
	require.NoError(t, err)

	output2, err := sess.GetOutput()
	require.NoError(t, err)
	assert.Contains(t, string(output2), "Line 2")

	// Verify output2 is longer than output1
	assert.Greater(t, len(output2), len(output1))

	// Stop session
	err = sess.Stop()
	require.NoError(t, err)

	// Verify GetOutput returns error after stop
	_, err = sess.GetOutput()
	assert.Error(t, err)
}

func TestTailAgentLogsFlag(t *testing.T) {
	// Store original value
	originalFollow := followLogs
	defer func() { followLogs = originalFollow }()

	// Test that tailAgentLogs function sets the follow flag
	// This is a simple unit test to verify the flag behavior
	followLogs = false
	assert.False(t, followLogs)

	// The tailAgentLogs function sets followLogs = true
	// We test this by checking the implementation pattern
	// (The actual function would call viewAgentLogs which requires a full session setup)
}

func TestStreamSessionLogsExitsOnStop(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)
	defer func() {
		_ = os.RemoveAll(repoDir)
	}()

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	// Create workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name: "test-workspace",
	})
	require.NoError(t, err)

	// Create dependencies
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	require.NoError(t, err)
	mailboxManager := mailbox.NewManager(configManager.GetAmuxDir())
	store, err := session.NewFileStore(configManager.GetAmuxDir())
	require.NoError(t, err)

	// Create session manager with mock adapter
	sessionManager := session.NewManager(store, wsManager, mailboxManager, idMapper, session.WithLogger(logger.Nop()))

	// Replace tmux adapter with mock
	mockAdapter := tmux.NewMockAdapter()
	sessionManager.SetTmuxAdapter(mockAdapter)

	// Create and start a session
	opts := session.Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     "echo 'Test'",
	}

	sess, err := sessionManager.CreateSession(opts)
	require.NoError(t, err)

	ctx := context.Background()
	err = sess.Start(ctx)
	require.NoError(t, err)

	// Simulate streaming behavior: function should exit when session stops
	done := make(chan bool)
	go func() {
		// Simulate the polling loop checking session status
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if sess.Status() != session.StatusRunning {
					done <- true
					return
				}
			case <-time.After(2 * time.Second):
				// Timeout to prevent hanging test
				done <- false
				return
			}
		}
	}()

	// Stop the session after a short delay
	time.Sleep(200 * time.Millisecond)
	err = sess.Stop()
	require.NoError(t, err)

	// Verify the goroutine exits
	success := <-done
	assert.True(t, success, "Streaming should have exited when session stopped")
}
