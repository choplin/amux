package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/runtime"
)

func TestLocalRuntime_Execute(t *testing.T) {
	tests := []struct {
		name    string
		spec    runtime.ExecutionSpec
		wantErr bool
		errType error
		check   func(t *testing.T, p runtime.Process)
	}{
		{
			name: "simple echo command",
			spec: runtime.ExecutionSpec{
				Command: []string{"echo", "hello"},
				Options: LocalOptions{
					CaptureOutput: true,
				},
			},
			check: func(t *testing.T, p runtime.Process) {
				assert.Equal(t, runtime.StateRunning, p.State())

				// Wait for completion
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)

				// Check output
				stdout, _ := p.Output()
				output, err := io.ReadAll(stdout)
				require.NoError(t, err)
				assert.Equal(t, "hello\n", string(output))

				// Check exit code
				code, err := p.ExitCode()
				require.NoError(t, err)
				assert.Equal(t, 0, code)
			},
		},
		{
			name: "command with environment variables",
			spec: runtime.ExecutionSpec{
				Command: []string{"sh", "-c", "echo $TEST_VAR"},
				Environment: map[string]string{
					"TEST_VAR": "test-value",
				},
				Options: LocalOptions{
					CaptureOutput: true,
				},
			},
			check: func(t *testing.T, p runtime.Process) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)

				stdout, _ := p.Output()
				output, err := io.ReadAll(stdout)
				require.NoError(t, err)
				assert.Equal(t, "test-value\n", string(output))
			},
		},
		{
			name: "command with working directory",
			spec: runtime.ExecutionSpec{
				Command:    []string{"pwd"},
				WorkingDir: "/tmp",
				Options: LocalOptions{
					CaptureOutput: true,
				},
			},
			check: func(t *testing.T, p runtime.Process) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)

				stdout, _ := p.Output()
				output, err := io.ReadAll(stdout)
				require.NoError(t, err)
				assert.Equal(t, "/tmp\n", string(output))
			},
		},
		{
			name: "empty command",
			spec: runtime.ExecutionSpec{
				Command: []string{},
			},
			wantErr: true,
			errType: runtime.ErrInvalidCommand,
		},
		{
			name: "non-existent working directory",
			spec: runtime.ExecutionSpec{
				Command:    []string{"echo", "test"},
				WorkingDir: "/non/existent/directory",
			},
			wantErr: true,
		},
		{
			name: "command with stderr output",
			spec: runtime.ExecutionSpec{
				Command: []string{"sh", "-c", "echo error >&2"},
				Options: LocalOptions{
					CaptureOutput: true,
				},
			},
			check: func(t *testing.T, p runtime.Process) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)

				_, stderr := p.Output()
				output, err := io.ReadAll(stderr)
				require.NoError(t, err)
				assert.Equal(t, "error\n", string(output))
			},
		},
		{
			name: "long running process",
			spec: runtime.ExecutionSpec{
				Command: []string{"sleep", "0.1"},
			},
			check: func(t *testing.T, p runtime.Process) {
				// Should be running initially
				assert.Equal(t, runtime.StateRunning, p.State())

				// Wait for completion
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)

				// Should be stopped
				assert.Equal(t, runtime.StateStopped, p.State())
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

func TestLocalRuntime_Find(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Create a process
	p1, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"sleep", "1"},
	})
	require.NoError(t, err)

	// Should find the process
	found, err := r.Find(ctx, p1.ID())
	require.NoError(t, err)
	assert.Equal(t, p1.ID(), found.ID())

	// Should not find non-existent process
	_, err = r.Find(ctx, "non-existent-id")
	assert.ErrorIs(t, err, runtime.ErrProcessNotFound)

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
	p1, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"sleep", "1"},
	})
	require.NoError(t, err)

	p2, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"sleep", "1"},
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
	r := New()
	ctx := context.Background()

	// Create a long-running process
	p, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"sleep", "10"},
	})
	require.NoError(t, err)

	// Should be running
	assert.Equal(t, runtime.StateRunning, p.State())

	// Stop the process
	err = p.Stop(ctx)
	require.NoError(t, err)

	// Should be stopped
	time.Sleep(100 * time.Millisecond) // Give it time to update state
	assert.NotEqual(t, runtime.StateRunning, p.State())

	// Stopping again should error
	err = p.Stop(ctx)
	assert.ErrorIs(t, err, runtime.ErrProcessAlreadyDone)
}

func TestLocalRuntime_Kill(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Create a process that ignores SIGTERM
	p, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"sh", "-c", "trap '' TERM; sleep 10"},
	})
	require.NoError(t, err)

	// Kill the process
	err = p.Kill(ctx)
	require.NoError(t, err)

	// Should be stopped
	time.Sleep(100 * time.Millisecond) // Give it time to update state
	assert.NotEqual(t, runtime.StateRunning, p.State())
}

func TestLocalRuntime_ContextCancellation(t *testing.T) {
	r := New()
	ctx, cancel := context.WithCancel(context.Background())

	// Create a long-running process
	p, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"sleep", "10"},
	})
	require.NoError(t, err)

	// Cancel the context
	cancel()

	// Process should be stopped
	time.Sleep(100 * time.Millisecond) // Give it time to react
	assert.NotEqual(t, runtime.StateRunning, p.State())
}

func TestLocalRuntime_OutputCapture(t *testing.T) {
	r := New()
	ctx := context.Background()

	t.Run("limited output", func(t *testing.T) {
		p, err := r.Execute(ctx, runtime.ExecutionSpec{
			Command: []string{"sh", "-c", "echo '1234567890' && echo '1234567890' && echo '1234567890' && echo '1234567890' && echo '1234567890' && echo '1234567890' && echo '1234567890' && echo '1234567890' && echo '1234567890' && echo '1234567890' && echo '1234567890'"},
			Options: LocalOptions{
				CaptureOutput:   true,
				OutputSizeLimit: 50, // Very small limit
			},
		})
		require.NoError(t, err)

		// Wait for completion
		err = p.Wait(context.Background())
		// The process may exit with broken pipe, which is expected
		// when output is limited

		// Output should be limited
		stdout, _ := p.Output()
		output, err := io.ReadAll(stdout)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(output), 50)
	})

	t.Run("no capture", func(t *testing.T) {
		p, err := r.Execute(ctx, runtime.ExecutionSpec{
			Command: []string{"echo", "test"},
			Options: LocalOptions{
				CaptureOutput: false,
			},
		})
		require.NoError(t, err)

		// Wait for completion
		err = p.Wait(context.Background())
		require.NoError(t, err)

		// No output should be captured
		stdout, stderr := p.Output()
		assert.Nil(t, stdout)
		assert.Nil(t, stderr)
	})
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
	p, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"false"},
	})
	require.NoError(t, err)

	// Wait for completion
	err = p.Wait(context.Background())
	require.Error(t, err)

	// Should be in failed state
	assert.Equal(t, runtime.StateFailed, p.State())

	// Exit code should be non-zero
	code, err := p.ExitCode()
	require.NoError(t, err)
	assert.NotEqual(t, 0, code)
}

func TestLocalRuntime_SingleCommandShell(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Test that single commands are run through shell
	p, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"echo hello && echo world"},
		Options: LocalOptions{
			CaptureOutput: true,
		},
	})
	require.NoError(t, err)

	// Wait for completion
	err = p.Wait(context.Background())
	require.NoError(t, err)

	// Should have both outputs
	stdout, _ := p.Output()
	output, err := io.ReadAll(stdout)
	require.NoError(t, err)
	assert.Contains(t, string(output), "hello")
	assert.Contains(t, string(output), "world")
}

func TestLocalRuntime_InheritEnv(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Set a test environment variable
	os.Setenv("TEST_INHERIT_VAR", "inherited-value")
	defer os.Unsetenv("TEST_INHERIT_VAR")

	// Execute with InheritEnv
	p, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"sh", "-c", "echo $TEST_INHERIT_VAR"},
		Options: LocalOptions{
			InheritEnv:    true,
			CaptureOutput: true,
		},
	})
	require.NoError(t, err)

	// Wait for completion
	err = p.Wait(context.Background())
	require.NoError(t, err)

	// Should have the inherited variable
	stdout, _ := p.Output()
	output, err := io.ReadAll(stdout)
	require.NoError(t, err)
	assert.Equal(t, "inherited-value\n", string(output))
}

func TestProcess_WaitTimeout(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Create a long-running process
	p, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"sleep", "10"},
	})
	require.NoError(t, err)

	// Wait with timeout
	waitCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = p.Wait(waitCtx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Clean up
	_ = p.Kill(ctx)
}

func TestProcess_ExitCodeWhileRunning(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Create a long-running process
	p, err := r.Execute(ctx, runtime.ExecutionSpec{
		Command: []string{"sleep", "1"},
	})
	require.NoError(t, err)

	// Try to get exit code while running
	_, err = p.ExitCode()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "still running")

	// Clean up
	_ = p.Kill(ctx)
}

func TestLocalOptions_RuntimeInterface(t *testing.T) {
	// Ensure LocalOptions implements RuntimeOptions
	var _ runtime.RuntimeOptions = LocalOptions{}
}

func TestLimitedBuffer(t *testing.T) {
	t.Run("respects limit", func(t *testing.T) {
		buf := &limitedBuffer{limit: 10}

		// Write more than limit
		n, err := buf.Write([]byte("1234567890ABC"))
		assert.NoError(t, err)
		assert.Equal(t, 10, n)

		// Buffer should only contain first 10 bytes
		assert.Equal(t, "1234567890", string(buf.Bytes()))

		// Writing more should not increase size
		n, err = buf.Write([]byte("XYZ"))
		assert.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.Equal(t, "1234567890", string(buf.Bytes()))
	})

	t.Run("no limit", func(t *testing.T) {
		buf := &limitedBuffer{limit: 0}

		// Should accept any amount
		data := strings.Repeat("x", 1000)
		n, err := buf.Write([]byte(data))
		assert.NoError(t, err)
		assert.Equal(t, 1000, n)
		assert.Equal(t, data, string(buf.Bytes()))
	})
}

func TestProcess_Concurrent(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Create multiple processes concurrently
	const numProcesses = 10
	processes := make([]runtime.Process, numProcesses)
	errors := make([]error, numProcesses)

	var wg sync.WaitGroup
	wg.Add(numProcesses)

	for i := 0; i < numProcesses; i++ {
		go func(idx int) {
			defer wg.Done()
			p, err := r.Execute(ctx, runtime.ExecutionSpec{
				Command: []string{"echo", fmt.Sprintf("process-%d", idx)},
				Options: LocalOptions{
					CaptureOutput: true,
				},
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
