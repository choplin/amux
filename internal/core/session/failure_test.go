package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/workspace"
)

// MockProcessChecker implements process.Checker for testing
type MockProcessChecker struct {
	hasChildren map[int]bool
}

func NewMockProcessChecker() *MockProcessChecker {
	return &MockProcessChecker{
		hasChildren: make(map[int]bool),
	}
}

func (m *MockProcessChecker) HasChildren(pid int) (bool, error) {
	if val, ok := m.hasChildren[pid]; ok {
		return val, nil
	}
	return true, nil // Default to having children
}

func (m *MockProcessChecker) SetHasChildren(pid int, hasChildren bool) {
	m.hasChildren[pid] = hasChildren
}

func TestSessionFailureDetection(t *testing.T) {
	// Skip if tmux not available
	tmuxAdapter, err := tmux.NewAdapter()
	if err != nil || !tmuxAdapter.IsAvailable() {
		t.Skip("tmux not available")
	}

	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create manager
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	t.Run("Session marked as failed when tmux session doesn't exist", func(t *testing.T) {
		// Create mock adapter
		mockAdapter := tmux.NewMockAdapter()

		// Create session info
		info := &Info{
			ID:          "test-session-1",
			WorkspaceID: ws.ID,
			AgentID:     "test-agent",
			StatusState: StatusState{
				Status:          StatusRunning,
				StatusChangedAt: time.Now(),
			},
			TmuxSession: "test-tmux-session",
			PID:         12345,
		}

		// Create session
		sess := CreateTmuxSession(context.Background(), info, manager, mockAdapter, ws, nil)

		// Create tmux session first
		err := mockAdapter.CreateSession(info.TmuxSession, ws.Path)
		require.NoError(t, err)

		// Verify session is running
		assert.Equal(t, StatusRunning, sess.Status())

		// Kill the tmux session
		err = mockAdapter.KillSession(info.TmuxSession)
		require.NoError(t, err)

		// Update status
		err = sess.UpdateStatus(context.Background())
		require.NoError(t, err)

		// Should be marked as failed
		assert.Equal(t, StatusFailed, sess.Status())
		assert.Equal(t, "tmux session no longer exists", sess.Info().Error)
	})

	t.Run("Session marked as failed when shell process is dead", func(t *testing.T) {
		// Create mock adapter
		mockAdapter := tmux.NewMockAdapter()

		// Create session info
		info := &Info{
			ID:          "test-session-2",
			WorkspaceID: ws.ID,
			AgentID:     "test-agent",
			StatusState: StatusState{
				Status:          StatusRunning,
				StatusChangedAt: time.Now(),
			},
			TmuxSession: "test-tmux-session-2",
			PID:         12346,
		}

		// Create session
		sess := CreateTmuxSession(context.Background(), info, manager, mockAdapter, ws, nil)

		// Create tmux session
		err := mockAdapter.CreateSession(info.TmuxSession, ws.Path)
		require.NoError(t, err)

		// Mark pane as dead
		err = mockAdapter.SetPaneDead(info.TmuxSession, true)
		require.NoError(t, err)

		// Update status
		err = sess.UpdateStatus(context.Background())
		require.NoError(t, err)

		// Should be marked as failed
		assert.Equal(t, StatusFailed, sess.Status())
		assert.Equal(t, "shell process exited", sess.Info().Error)
	})

	t.Run("Session marked as completed when no child processes", func(t *testing.T) {
		// Create mock adapter and process checker
		mockAdapter := tmux.NewMockAdapter()
		mockProcessChecker := NewMockProcessChecker()

		// Create session info
		info := &Info{
			ID:          "test-session-completed",
			WorkspaceID: ws.ID,
			AgentID:     "test-agent",
			StatusState: StatusState{
				Status:          StatusRunning,
				StatusChangedAt: time.Now(),
			},
			TmuxSession: "test-tmux-session-completed",
			PID:         12348,
		}

		// Create session with mock process checker
		sess := CreateTmuxSession(context.Background(), info, manager, mockAdapter, ws, nil, WithProcessChecker(mockProcessChecker))

		// Create tmux session
		err := mockAdapter.CreateSession(info.TmuxSession, ws.Path)
		require.NoError(t, err)

		// Set process to have no children
		mockProcessChecker.SetHasChildren(info.PID, false)

		// Update status
		err = sess.UpdateStatus(context.Background())
		require.NoError(t, err)

		// Should be marked as completed
		assert.Equal(t, StatusCompleted, sess.Status())
		assert.Empty(t, sess.Info().Error)
	})

	t.Run("Session remains working when all checks pass", func(t *testing.T) {
		// Create mock adapter and process checker
		mockAdapter := tmux.NewMockAdapter()
		mockProcessChecker := NewMockProcessChecker()

		// Create session info
		info := &Info{
			ID:          "test-session-3",
			WorkspaceID: ws.ID,
			AgentID:     "test-agent",
			StatusState: StatusState{
				Status:          StatusWorking,
				StatusChangedAt: time.Now(),
				LastOutputHash:  12345,
				LastOutputTime:  time.Now(),
			},
			TmuxSession: "test-tmux-session-3",
			PID:         12347,
		}

		// Create session with mock process checker
		sess := CreateTmuxSession(context.Background(), info, manager, mockAdapter, ws, nil, WithProcessChecker(mockProcessChecker))

		// Create tmux session
		err := mockAdapter.CreateSession(info.TmuxSession, ws.Path)
		require.NoError(t, err)

		// Set some output
		mockAdapter.SetPaneContent(info.TmuxSession, "test output")

		// Process has children (default behavior)
		mockProcessChecker.SetHasChildren(info.PID, true)

		// Update status
		err = sess.UpdateStatus(context.Background())
		require.NoError(t, err)

		// Should remain working
		assert.Equal(t, StatusWorking, sess.Status())
		assert.Empty(t, sess.Info().Error)
	})

	t.Run("Session marked as failed when exit status is non-zero", func(t *testing.T) {
		// Create mock adapter and process checker
		mockAdapter := tmux.NewMockAdapter()
		mockProcessChecker := NewMockProcessChecker()

		// Create temp storage directory
		storageDir := t.TempDir()

		// Create session info with storage path
		info := &Info{
			ID:          "test-session-exit-failed",
			WorkspaceID: ws.ID,
			AgentID:     "test-agent",
			StatusState: StatusState{
				Status:          StatusRunning,
				StatusChangedAt: time.Now(),
			},
			TmuxSession: "test-tmux-session-exit-failed",
			PID:         12350,
			StoragePath: storageDir,
		}

		// Create session with mock process checker
		sess := CreateTmuxSession(context.Background(), info, manager, mockAdapter, ws, nil, WithProcessChecker(mockProcessChecker))

		// Create tmux session
		err := mockAdapter.CreateSession(info.TmuxSession, ws.Path)
		require.NoError(t, err)

		// Pre-create exit status file since mock doesn't execute shell commands
		exitStatusPath := filepath.Join(storageDir, "exit_status")
		err = os.WriteFile(exitStatusPath, []byte("1\n"), 0o644)
		require.NoError(t, err)

		// Set process to have no children
		mockProcessChecker.SetHasChildren(info.PID, false)

		// Update status
		err = sess.UpdateStatus(context.Background())
		require.NoError(t, err)

		// Should be marked as failed
		assert.Equal(t, StatusFailed, sess.Status())
		assert.Equal(t, "command exited with code 1", sess.Info().Error)
	})

	t.Run("Session marked as completed when exit status is zero", func(t *testing.T) {
		// Create mock adapter and process checker
		mockAdapter := tmux.NewMockAdapter()
		mockProcessChecker := NewMockProcessChecker()

		// Create temp storage directory
		storageDir := t.TempDir()

		// Create session info with storage path
		info := &Info{
			ID:          "test-session-exit-success",
			WorkspaceID: ws.ID,
			AgentID:     "test-agent",
			StatusState: StatusState{
				Status:          StatusRunning,
				StatusChangedAt: time.Now(),
			},
			TmuxSession: "test-tmux-session-exit-success",
			PID:         12351,
			StoragePath: storageDir,
		}

		// Create session with mock process checker
		sess := CreateTmuxSession(context.Background(), info, manager, mockAdapter, ws, nil, WithProcessChecker(mockProcessChecker))

		// Create tmux session
		err := mockAdapter.CreateSession(info.TmuxSession, ws.Path)
		require.NoError(t, err)

		// Pre-create exit status file since mock doesn't execute shell commands
		exitStatusPath := filepath.Join(storageDir, "exit_status")
		err = os.WriteFile(exitStatusPath, []byte("0\n"), 0o644)
		require.NoError(t, err)

		// Set process to have no children
		mockProcessChecker.SetHasChildren(info.PID, false)

		// Update status
		err = sess.UpdateStatus(context.Background())
		require.NoError(t, err)

		// Should be marked as completed
		assert.Equal(t, StatusCompleted, sess.Status())
		assert.Empty(t, sess.Info().Error)
	})

	t.Run("Session with running command transitions to completed", func(t *testing.T) {
		// Create mock adapter and process checker
		mockAdapter := tmux.NewMockAdapter()
		mockProcessChecker := NewMockProcessChecker()

		// Create session info
		info := &Info{
			ID:          "test-session-transition",
			WorkspaceID: ws.ID,
			AgentID:     "test-agent",
			StatusState: StatusState{
				Status:          StatusRunning,
				StatusChangedAt: time.Now(),
			},
			TmuxSession: "test-tmux-session-transition",
			PID:         12349,
		}

		// Create session with mock process checker
		sess := CreateTmuxSession(context.Background(), info, manager, mockAdapter, ws, nil, WithProcessChecker(mockProcessChecker))

		// Create tmux session
		err := mockAdapter.CreateSession(info.TmuxSession, ws.Path)
		require.NoError(t, err)

		// Initially has children
		mockProcessChecker.SetHasChildren(info.PID, true)

		// Reset cache to ensure update runs
		if tmuxSess, ok := sess.(*tmuxSessionImpl); ok {
			tmuxSess.info.StatusState.LastStatusCheck = time.Time{}
		}

		// Update status - should remain running
		err = sess.UpdateStatus(context.Background())
		require.NoError(t, err)
		assert.Equal(t, StatusRunning, sess.Status())

		// Now simulate command completion - no more children
		mockProcessChecker.SetHasChildren(info.PID, false)

		// Reset cache again to ensure update runs
		if tmuxSess, ok := sess.(*tmuxSessionImpl); ok {
			tmuxSess.info.StatusState.LastStatusCheck = time.Time{}
		}

		// Update status - should transition to completed
		err = sess.UpdateStatus(context.Background())
		require.NoError(t, err)
		assert.Equal(t, StatusCompleted, sess.Status())
	})
}
