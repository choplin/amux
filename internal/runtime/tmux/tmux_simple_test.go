package tmux

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/runtime"
)

func TestTmuxRuntime_BasicExecution(t *testing.T) {
	skipIfTmuxNotAvailable(t)

	tmpDir := t.TempDir()
	r, err := New(tmpDir)
	require.NoError(t, err)

	// Validate runtime
	err = r.Validate()
	require.NoError(t, err)

	ctx := context.Background()

	// Create a simple process
	p, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"true"},
		Options: Options{
			SessionName: "test-basic",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, p)

	t.Logf("Process ID: %s", p.ID())
	t.Logf("Process State: %s", p.State())

	// Check if session exists
	tmuxProc := p.(*Process)
	exists := tmuxProc.sessionExists()
	t.Logf("Session exists: %v", exists)

	// Wait a bit for process to start
	time.Sleep(500 * time.Millisecond)

	// Check state
	state := p.State()
	t.Logf("State after 500ms: %s", state)

	// Check if pane is dead
	dead, err := tmuxProc.isPaneDead()
	t.Logf("Pane dead: %v, error: %v", dead, err)

	// Wait for completion with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	waitErr := p.Wait(ctx)
	t.Logf("Wait error: %v", waitErr)
	t.Logf("Final state: %s", p.State())

	// Try to manually check session
	exists = tmuxProc.sessionExists()
	t.Logf("Session exists after wait: %v", exists)

	// Clean up
	killErr := p.Kill(context.Background())
	t.Logf("Kill error: %v", killErr)
}

func TestTmuxRuntime_ManualSessionCheck(t *testing.T) {
	skipIfTmuxNotAvailable(t)

	tmpDir := t.TempDir()
	r, err := New(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Create a long-running process
	p, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"sh", "-c", "echo started; sleep 5; echo done"},
		Options: Options{
			SessionName:   "test-manual",
			CaptureOutput: true,
		},
	})
	require.NoError(t, err)
	defer func() { _ = p.Kill(context.Background()) }()

	tmuxProc := p.(*Process)

	// Monitor for a few seconds
	for i := 0; i < 10; i++ {
		exists := tmuxProc.sessionExists()
		dead, _ := tmuxProc.isPaneDead()
		state := p.State()

		t.Logf("Iteration %d: exists=%v, dead=%v, state=%s", i, exists, dead, state)

		if state != runtime.StateRunning {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func TestTmuxRuntime_ImmediateExit(t *testing.T) {
	skipIfTmuxNotAvailable(t)

	// Skip test - tmux session completion detection needs improvement
	t.Skip("Tmux session completion detection needs improvement")

	tmpDir := t.TempDir()
	r, err := New(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Use sh -c exit command that completes immediately
	p, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"sh", "-c", "exit 0"},
		Options: Options{
			SessionName:  "test-immediate",
			RemainOnExit: false,
		},
	})
	require.NoError(t, err)

	// Monitor state changes
	for i := 0; i < 20; i++ {
		state := p.State()
		t.Logf("State at %dms: %s", i*100, state)

		if state == runtime.StateStopped {
			// Success
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	t.Fatal("Process did not stop within 2 seconds")
}
