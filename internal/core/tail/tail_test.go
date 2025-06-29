package tail_test

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/tail"
	"github.com/aki/amux/internal/core/workspace"
	runtimeinit "github.com/aki/amux/internal/runtime/init"
	"github.com/aki/amux/internal/tests/helpers"
)

// writerFunc is an adapter to allow the use of ordinary functions as io.Writer
type writerFunc struct {
	fn func(p []byte) (int, error)
}

func (w *writerFunc) Write(p []byte) (int, error) {
	return w.fn(p)
}

func TestTailer_Follow(t *testing.T) {
	// Initialize runtime registry for tests
	if err := runtimeinit.RegisterDefaults(); err != nil {
		t.Fatalf("Failed to initialize runtimes: %v", err)
	}

	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	cfg.Agents["test-agent"] = config.Agent{
		Name:    "Test Agent",
		Runtime: "tmux",
		Command: []string{"echo", "test"},
	}
	cfg.Agents["test-agent-2"] = config.Agent{
		Name:    "Test Agent 2",
		Runtime: "tmux",
		Command: []string{"echo", "test"},
	}
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	// Create workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace",
	})
	require.NoError(t, err)

	// Create dependencies
	idMapper, err := idmap.NewSessionIDMapper(configManager.GetAmuxDir())
	require.NoError(t, err)
	// Create session manager with mock adapter
	sessionManager, err := session.NewManager(configManager.GetAmuxDir(), wsManager, configManager, idMapper)
	require.NoError(t, err)

	// Replace tmux adapter with mock
	mockAdapter := tmux.NewMockAdapter()
	sessionManager.SetTmuxAdapter(mockAdapter)

	// Create and start a session
	opts := session.Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     "echo 'Starting test'",
	}

	sess, err := sessionManager.CreateSession(context.Background(), opts)
	require.NoError(t, err)

	ctx := context.Background()
	err = sess.Start(ctx)
	require.NoError(t, err)

	// Add initial output
	info := sess.Info()
	err = mockAdapter.AppendSessionOutput(info.TmuxSession, "Initial output\n")
	require.NoError(t, err)

	t.Run("follows new output", func(t *testing.T) {
		// Use a thread-safe buffer wrapper
		type safeBuffer struct {
			mu  sync.Mutex
			buf bytes.Buffer
		}
		safeBuf := &safeBuffer{}

		// Create a writer that locks the buffer
		writer := &writerFunc{fn: func(p []byte) (int, error) {
			safeBuf.mu.Lock()
			defer safeBuf.mu.Unlock()
			return safeBuf.buf.Write(p)
		}}

		// Create tailer with short poll interval for testing
		tailOpts := tail.Options{
			PollInterval: 100 * time.Millisecond,
			Writer:       writer,
		}
		tailer := tail.New(sess, tailOpts)

		// Start following in goroutine
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error)
		go func() {
			done <- tailer.Follow(ctx)
		}()

		// Helper to safely read the buffer
		getOutput := func() string {
			safeBuf.mu.Lock()
			defer safeBuf.mu.Unlock()
			return safeBuf.buf.String()
		}

		// Wait for initial output
		time.Sleep(150 * time.Millisecond)
		assert.Contains(t, getOutput(), "Initial output")

		// Add new output while following
		err = mockAdapter.AppendSessionOutput(info.TmuxSession, "New line 1\n")
		require.NoError(t, err)
		time.Sleep(150 * time.Millisecond)

		err = mockAdapter.AppendSessionOutput(info.TmuxSession, "New line 2\n")
		require.NoError(t, err)
		time.Sleep(150 * time.Millisecond)

		// Cancel context to stop following
		cancel()

		// Wait for Follow to exit
		select {
		case err := <-done:
			assert.ErrorIs(t, err, context.Canceled)
		case <-time.After(1 * time.Second):
			t.Fatal("Follow did not exit after context cancel")
		}

		// Check output
		output := getOutput()
		assert.Contains(t, output, "Initial output")
		assert.Contains(t, output, "New line 1")
		assert.Contains(t, output, "New line 2")
	})

	t.Run("exits when session stops", func(t *testing.T) {
		// Create new session for this test
		sess2, err := sessionManager.CreateSession(context.Background(), session.Options{
			WorkspaceID: ws.ID,
			AgentID:     "test-agent-2",
			Command:     "echo 'Test 2'",
		})
		require.NoError(t, err)

		err = sess2.Start(context.Background())
		require.NoError(t, err)

		var buf bytes.Buffer
		tailOpts := tail.Options{
			PollInterval: 100 * time.Millisecond,
			Writer:       &buf,
		}
		tailer := tail.New(sess2, tailOpts)

		// Start following
		done := make(chan error)
		go func() {
			done <- tailer.Follow(context.Background())
		}()

		// Stop the session after a short delay
		time.Sleep(200 * time.Millisecond)
		err = sess2.Stop(context.Background())
		require.NoError(t, err)

		// Follow should exit normally
		select {
		case err := <-done:
			assert.NoError(t, err) // Should exit normally, not with error
		case <-time.After(1 * time.Second):
			t.Fatal("Follow did not exit after session stopped")
		}
	})
}

func TestFollowFunc(t *testing.T) {
	// Initialize runtime registry for tests
	if err := runtimeinit.RegisterDefaults(); err != nil {
		t.Fatalf("Failed to initialize runtimes: %v", err)
	}

	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	cfg.Agents["test-agent"] = config.Agent{
		Name:    "Test Agent",
		Runtime: "tmux",
		Command: []string{"echo", "test"},
	}
	cfg.Agents["test-agent-2"] = config.Agent{
		Name:    "Test Agent 2",
		Runtime: "tmux",
		Command: []string{"echo", "test"},
	}
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	// Create workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace",
	})
	require.NoError(t, err)

	// Create dependencies
	idMapper, err := idmap.NewSessionIDMapper(configManager.GetAmuxDir())
	require.NoError(t, err)
	// Create session manager with mock adapter
	sessionManager, err := session.NewManager(configManager.GetAmuxDir(), wsManager, configManager, idMapper)
	require.NoError(t, err)

	// Replace tmux adapter with mock
	mockAdapter := tmux.NewMockAdapter()
	sessionManager.SetTmuxAdapter(mockAdapter)

	// Create and start a session
	opts := session.Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     "echo 'Test'",
	}

	sess, err := sessionManager.CreateSession(context.Background(), opts)
	require.NoError(t, err)

	ctx := context.Background()
	err = sess.Start(ctx)
	require.NoError(t, err)

	// Add some output
	info := sess.Info()
	err = mockAdapter.AppendSessionOutput(info.TmuxSession, "Test output\n")
	require.NoError(t, err)

	// Test FollowFunc
	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error)
	go func() {
		done <- tail.FollowFunc(ctx, sess, &buf)
	}()

	// Wait a bit and cancel
	time.Sleep(200 * time.Millisecond)
	cancel()

	// Should exit with context error
	err = <-done
	assert.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, buf.String(), "Test output")
}
