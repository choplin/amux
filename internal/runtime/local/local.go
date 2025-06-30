// Package local provides a runtime implementation that executes processes directly using os/exec.
package local

import (
	"context"
	"fmt"
	"os"
	"time"

	amuxruntime "github.com/aki/amux/internal/runtime"
)

// Runtime implements the local process runtime for foreground execution
type Runtime struct {
	baseRuntime
}

// New creates a new local runtime
func New() *Runtime {
	return &Runtime{}
}

// Type returns the runtime type identifier
func (r *Runtime) Type() string {
	return "local"
}

// Execute starts a new process in the local runtime (foreground mode)
func (r *Runtime) Execute(ctx context.Context, spec amuxruntime.ExecutionSpec) (amuxruntime.Process, error) {
	// Validate command
	if len(spec.Command) == 0 {
		return nil, amuxruntime.ErrInvalidCommand
	}

	// Get shell
	shell := getShell()

	// Create command with context for proper cancellation
	cmd := createCommand(ctx, spec, shell, true)

	// Setup command properties
	if err := setupCommand(cmd, spec); err != nil {
		return nil, err
	}

	// Configure process isolation for foreground
	configureProcessIsolation(cmd, false)

	// Create process
	proc := createProcess(spec)
	proc.cmd = cmd

	// For foreground processes, inherit stdout/stderr for real-time output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	proc.setState(amuxruntime.StateRunning)

	// Create metadata after process starts
	proc.metadata = &Metadata{
		PID:      cmd.Process.Pid,
		Detached: false,
	}

	// Try to get PGID (might not work on all platforms)
	if isProcessGroup(cmd) {
		proc.metadata.PGID = cmd.Process.Pid
	}

	// Store process
	r.processes.Store(proc.id, proc)

	// Monitor process completion
	go proc.monitor()

	// Handle context cancellation
	go func() {
		select {
		case <-ctx.Done():
			// Create a new context with timeout for cleanup
			stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = proc.Stop(stopCtx)
		case <-proc.done:
		}
	}()

	return proc, nil
}
