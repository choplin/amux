package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func getStderrCommand(text string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", "echo " + text + " 1>&2"}
	}
	return []string{"sh", "-c", "echo " + text + " >&2"}
}

func getSleepCommand(seconds float64) []string {
	if runtime.GOOS == "windows" {
		// Use PowerShell's Start-Sleep for more reliable behavior
		return []string{"powershell", "-Command", fmt.Sprintf("Start-Sleep -Seconds %.1f", seconds)}
	}
	return []string{"sleep", fmt.Sprintf("%.1f", seconds)}
}

func getShellCommand(cmd string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", cmd}
	}
	return []string{"sh", "-c", cmd}
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
			name: "simple echo command",
			spec: amuxruntime.ExecutionSpec{
				Command: []string{"echo", "hello"},
				Options: Options{
					CaptureOutput: true,
				},
			},
			check: func(t *testing.T, p amuxruntime.Process) {
				assert.Equal(t, amuxruntime.StateRunning, p.State())

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
			spec: amuxruntime.ExecutionSpec{
				Command: getPrintEnvCommand("TEST_VAR"),
				Environment: map[string]string{
					"TEST_VAR": "test-value",
				},
				Options: Options{
					CaptureOutput: true,
				},
			},
			check: func(t *testing.T, p amuxruntime.Process) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)

				stdout, _ := p.Output()
				output, err := io.ReadAll(stdout)
				require.NoError(t, err)
				assert.Contains(t, string(output), "test-value")
			},
		},
		{
			name: "command with working directory",
			spec: amuxruntime.ExecutionSpec{
				Command:    getPwdCommand(),
				WorkingDir: os.TempDir(),
				Options: Options{
					CaptureOutput: true,
				},
			},
			check: func(t *testing.T, p amuxruntime.Process) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)

				stdout, _ := p.Output()
				output, err := io.ReadAll(stdout)
				require.NoError(t, err)
				// On Windows, paths might have different formats
				assert.Contains(t, string(output), filepath.Base(os.TempDir()))
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
			name: "command with stderr output",
			spec: amuxruntime.ExecutionSpec{
				Command: getStderrCommand("error"),
				Options: Options{
					CaptureOutput: true,
				},
			},
			check: func(t *testing.T, p amuxruntime.Process) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				err := p.Wait(ctx)
				require.NoError(t, err)

				_, stderr := p.Output()
				output, err := io.ReadAll(stderr)
				require.NoError(t, err)
				assert.Contains(t, string(output), "error")
			},
		},
		{
			name: "long running process",
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

	// Create a long-running process
	p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: getSleepCommand(10),
	})
	require.NoError(t, err)

	// Cancel the context
	cancel()

	// Process should be stopped
	time.Sleep(100 * time.Millisecond) // Give it time to react
	assert.NotEqual(t, amuxruntime.StateRunning, p.State())
}

func TestLocalRuntime_OutputCapture(t *testing.T) {
	r := New()
	ctx := context.Background()

	t.Run("limited output", func(t *testing.T) {
		p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
			Command: getShellCommand("echo 1234567890 && echo 1234567890 && echo 1234567890 && echo 1234567890 && echo 1234567890 && echo 1234567890 && echo 1234567890 && echo 1234567890 && echo 1234567890 && echo 1234567890 && echo 1234567890"),
			Options: Options{
				CaptureOutput:   true,
				OutputSizeLimit: 50, // Very small limit
			},
		})
		require.NoError(t, err)

		// Wait for completion
		waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = p.Wait(waitCtx)
		// The process may exit with broken pipe, which is expected
		// when output is limited

		// Output should be limited
		stdout, _ := p.Output()
		output, err := io.ReadAll(stdout)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(output), 50)
	})

	t.Run("no capture", func(t *testing.T) {
		p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
			Command: []string{"echo", "test"},
			Options: Options{
				CaptureOutput: false,
			},
		})
		require.NoError(t, err)

		// Wait for completion
		waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		err = p.Wait(waitCtx)
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

func TestLocalRuntime_SingleCommandShell(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Test that single commands are run through shell
	cmd := "echo hello && echo world"
	if runtime.GOOS == "windows" {
		cmd = "echo hello & echo world"
	}

	p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: []string{cmd},
		Options: Options{
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
	p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: getPrintEnvCommand("TEST_INHERIT_VAR"),
		Options: Options{
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
	assert.Contains(t, string(output), "inherited-value")
}

func TestProcess_WaitTimeout(t *testing.T) {
	r := New()
	ctx := context.Background()

	// Create a long-running process
	p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
		Command: getSleepCommand(10),
	})
	require.NoError(t, err)

	// Wait with timeout
	waitCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = p.Wait(waitCtx)
	// On Windows, the process might exit immediately or timeout
	if err != nil {
		// Either context deadline exceeded or exit status error is acceptable
		assert.True(t, errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "exit status"),
			"Expected timeout or exit error, got: %v", err)
	}

	// Clean up
	_ = p.Kill(ctx)
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
	processes := make([]amuxruntime.Process, numProcesses)
	errors := make([]error, numProcesses)

	var wg sync.WaitGroup
	wg.Add(numProcesses)

	for i := 0; i < numProcesses; i++ {
		go func(idx int) {
			defer wg.Done()
			p, err := r.Execute(ctx, amuxruntime.ExecutionSpec{
				Command: []string{"echo", fmt.Sprintf("process-%d", idx)},
				Options: Options{
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
