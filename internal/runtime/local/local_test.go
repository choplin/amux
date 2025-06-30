package local

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	amuxruntime "github.com/aki/amux/internal/runtime"
)

// Helper functions for cross-platform commands
func getPrintEnvCommand(envVar string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "echo %" + envVar + "%"}
	}
	return []string{"sh", "-c", "echo $" + envVar}
}

func getSleepCommand(seconds float64) []string {
	if runtime.GOOS == "windows" {
		// Use PowerShell's Start-Sleep for more reliable behavior
		return []string{"powershell", "-Command", fmt.Sprintf("Start-Sleep -Seconds %.1f", seconds)}
	}
	return []string{"sleep", fmt.Sprintf("%.1f", seconds)}
}

func getPwdCommand() []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "cd"}
	}
	return []string{"pwd"}
}

func TestLocalRuntime_Execute(t *testing.T) {
	tests := []struct {
		name    string
		spec    amuxruntime.ExecutionSpec
		wantErr bool
		errType error
		check   func(t *testing.T, p amuxruntime.Process)
	}{
		{
			name: "simple echo command (foreground)",
			spec: amuxruntime.ExecutionSpec{
				Command: []string{"echo", "hello"},
			},
			check: func(t *testing.T, p amuxruntime.Process) {
				assert.Equal(t, amuxruntime.StateRunning, p.State())

				// Wait for completion
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)

				// Check exit code
				code, err := p.ExitCode()
				require.NoError(t, err)
				assert.Equal(t, 0, code)

				// Check metadata
				meta, ok := p.Metadata().(*Metadata)
				require.True(t, ok)
				assert.False(t, meta.Detached)
			},
		},
		{
			name: "command with environment variables",
			spec: amuxruntime.ExecutionSpec{
				Command: getPrintEnvCommand("TEST_VAR"),
				Environment: map[string]string{
					"TEST_VAR": "test-value",
				},
			},
			check: func(t *testing.T, p amuxruntime.Process) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)
			},
		},
		{
			name: "command with working directory",
			spec: amuxruntime.ExecutionSpec{
				Command:    getPwdCommand(),
				WorkingDir: os.TempDir(),
			},
			check: func(t *testing.T, p amuxruntime.Process) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)
			},
		},
		{
			name: "empty command",
			spec: amuxruntime.ExecutionSpec{
				Command: []string{},
			},
			wantErr: true,
			errType: amuxruntime.ErrInvalidCommand,
		},
		{
			name: "non-existent working directory",
			spec: amuxruntime.ExecutionSpec{
				Command:    []string{"echo", "test"},
				WorkingDir: "/non/existent/directory",
			},
			wantErr: true,
		},
		{
			name: "long running process (foreground)",
			spec: amuxruntime.ExecutionSpec{
				Command: getSleepCommand(0.1),
			},
			check: func(t *testing.T, p amuxruntime.Process) {
				// Should be running initially
				assert.Equal(t, amuxruntime.StateRunning, p.State())

				// Wait for completion
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)

				// Should be stopped
				assert.Equal(t, amuxruntime.StateStopped, p.State())
			},
		},
		{
			name: "single command through shell",
			spec: amuxruntime.ExecutionSpec{
				Command: []string{"echo hello && echo world"},
			},
			check: func(t *testing.T, p amuxruntime.Process) {
				// Wait for completion
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New()
			ctx := context.Background()

			p, err := r.Execute(ctx, tt.spec)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errType != nil {
					assert.ErrorIs(t, err, tt.errType)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, p)

			// Check basic properties
			assert.NotEmpty(t, p.ID())
			assert.False(t, p.StartTime().IsZero())

			if tt.check != nil {
				tt.check(t, p)
			}
		})
	}
}

func TestDetachedRuntime_Execute(t *testing.T) {
	r := NewDetachedRuntime()
	ctx := context.Background()

	// Create a detached process
	p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: getSleepCommand(0.5),
	})
	require.NoError(t, err)

	// Should be running
	assert.Equal(t, amuxruntime.StateRunning, p.State())

	// Check metadata
	meta, ok := p.Metadata().(*Metadata)
	require.True(t, ok)
	assert.True(t, meta.Detached)

	// Process should continue running even after short delay
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, amuxruntime.StateRunning, p.State())

	// Wait for natural completion
	ctx2, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = p.Wait(ctx2)
	require.NoError(t, err)

	// Should be stopped
	assert.Equal(t, amuxruntime.StateStopped, p.State())
}

func TestLocalRuntime_Find(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Create a process
	p1, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: getSleepCommand(1),
	})
	require.NoError(t, err)

	// Should find the process
	found, err := r.Find(ctx, p1.ID())
	require.NoError(t, err)
	assert.Equal(t, p1.ID(), found.ID())

	// Should not find non-existent process
	_, err = r.Find(ctx, "non-existent-id")
	assert.ErrorIs(t, err, amuxruntime.ErrProcessNotFound)

	// Clean up
	_ = p1.Kill(ctx)
}

func TestLocalRuntime_List(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Initially empty
	processes, err := r.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, processes)

	// Create multiple processes
	p1, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: getSleepCommand(1),
	})
	require.NoError(t, err)

	p2, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: getSleepCommand(1),
	})
	require.NoError(t, err)

	// Should list both processes
	processes, err = r.List(ctx)
	require.NoError(t, err)
	assert.Len(t, processes, 2)

	// Check that both processes are in the list
	ids := make(map[string]bool)
	for _, p := range processes {
		ids[p.ID()] = true
	}
	assert.True(t, ids[p1.ID()])
	assert.True(t, ids[p2.ID()])

	// Clean up
	_ = p1.Kill(ctx)
	_ = p2.Kill(ctx)
}

func TestLocalRuntime_Stop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM is not supported on Windows")
	}

	r := New()
	ctx := context.Background()

	// Create a long-running process
	p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: getSleepCommand(10),
	})
	require.NoError(t, err)

	// Should be running
	assert.Equal(t, amuxruntime.StateRunning, p.State())

	// Stop the process
	err = p.Stop(ctx)
	require.NoError(t, err)

	// Should be stopped
	time.Sleep(100 * time.Millisecond) // Give it time to update state
	assert.NotEqual(t, amuxruntime.StateRunning, p.State())

	// Stopping again should error
	err = p.Stop(ctx)
	assert.ErrorIs(t, err, amuxruntime.ErrProcessAlreadyDone)
}

func TestLocalRuntime_Kill(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Create a long-running process
	p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: getSleepCommand(10),
	})
	require.NoError(t, err)

	// Kill the process
	err = p.Kill(ctx)
	require.NoError(t, err)

	// Should be stopped
	time.Sleep(100 * time.Millisecond) // Give it time to update state
	assert.NotEqual(t, amuxruntime.StateRunning, p.State())
}

func TestLocalRuntime_ContextCancellation(t *testing.T) {
	r := New()
	ctx, cancel := context.WithCancel(context.Background())

	// Create a foreground process
	p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: getSleepCommand(10),
	})
	require.NoError(t, err)

	// Cancel the context
	cancel()

	// Process should be stopped (for foreground processes)
	time.Sleep(200 * time.Millisecond) // Give it time to react
	assert.NotEqual(t, amuxruntime.StateRunning, p.State())
}

func TestDetachedRuntime_ContextCancellation(t *testing.T) {
	r := NewDetachedRuntime()
	ctx, cancel := context.WithCancel(context.Background())

	// Create a detached process
	p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: getSleepCommand(1),
	})
	require.NoError(t, err)

	// Cancel the context immediately
	cancel()

	// Detached process should continue running
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, amuxruntime.StateRunning, p.State())

	// Wait for natural completion
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer waitCancel()
	err = p.Wait(waitCtx)
	require.NoError(t, err)

	// Should complete normally
	assert.Equal(t, amuxruntime.StateStopped, p.State())
}

func TestLocalRuntime_ProcessGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Process group testing is Unix-specific")
	}

	r := New()
	ctx := context.Background()

	// Create a process that spawns children
	shellCmd := `sh -c 'sleep 10 & sleep 10 & wait'`
	p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: []string{shellCmd},
	})
	require.NoError(t, err)

	// Give it time to spawn children
	time.Sleep(100 * time.Millisecond)

	// Kill the process (should kill the entire process group)
	err = p.Kill(ctx)
	require.NoError(t, err)

	// All processes in the group should be terminated
	time.Sleep(100 * time.Millisecond)
	assert.NotEqual(t, amuxruntime.StateRunning, p.State())
}

func TestLocalRuntime_Validate(t *testing.T) {
	r := New()
	// Local runtime should always be valid
	assert.NoError(t, r.Validate())
}

func TestLocalRuntime_FailedCommand(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Execute a command that will fail
	var failCmd []string
	if runtime.GOOS == "windows" {
		failCmd = []string{"cmd", "/c", "exit 1"}
	} else {
		failCmd = []string{"false"}
	}

	p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: failCmd,
	})
	require.NoError(t, err)

	// Wait for completion
	err = p.Wait(context.Background())
	require.Error(t, err)

	// Should be in failed state
	assert.Equal(t, amuxruntime.StateFailed, p.State())

	// Exit code should be non-zero
	code, err := p.ExitCode()
	require.NoError(t, err)
	assert.NotEqual(t, 0, code)
}

func TestProcess_ExitCodeWhileRunning(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Create a long-running process
	p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: getSleepCommand(1),
	})
	require.NoError(t, err)

	// Try to get exit code while running
	_, err = p.ExitCode()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "still running")

	// Clean up
	_ = p.Kill(ctx)
}

func TestOptions_RuntimeInterface(t *testing.T) {
	// Ensure Options implements RuntimeOptions
	var _ amuxruntime.RuntimeOptions = Options{}
}

func TestProcess_Concurrent(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Create multiple processes concurrently
	const numProcesses = 10
	processes := make([]amuxruntime.Process, numProcesses)
	errors := make([]error, numProcesses)

	var wg sync.WaitGroup
	wg.Add(numProcesses)

	for i := 0; i < numProcesses; i++ {
		go func(idx int) {
			defer wg.Done()
			p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
				Command: []string{"echo", fmt.Sprintf("process-%d", idx)},
			})
			processes[idx] = p
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// All should succeed
	for i, err := range errors {
		require.NoError(t, err, "process %d failed", i)
	}

	// All should have unique IDs
	ids := make(map[string]bool)
	for _, p := range processes {
		assert.NotContains(t, ids, p.ID())
		ids[p.ID()] = true
	}

	// Wait for all to complete
	for _, p := range processes {
		_ = p.Wait(context.Background())
	}
}

func TestDetachedRuntime_Type(t *testing.T) {
	r := NewDetachedRuntime()
	assert.Equal(t, "local-detached", r.Type())
}

func TestLocalRuntime_Type(t *testing.T) {
	r := New()
	assert.Equal(t, "local", r.Type())
}
