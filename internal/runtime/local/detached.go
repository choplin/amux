package local

import (
	"context"
	"fmt"

	amuxruntime "github.com/aki/amux/internal/runtime"
)

// DetachedRuntime implements the local process runtime for background/detached execution
type DetachedRuntime struct {
	baseRuntime
}

// NewDetachedRuntime creates a new detached runtime
func NewDetachedRuntime() *DetachedRuntime {
	return &DetachedRuntime{}
}

// Type returns the runtime type identifier
func (r *DetachedRuntime) Type() string {
	return "local-detached"
}

// Execute starts a new process in detached mode
func (r *DetachedRuntime) Execute(ctx context.Context, spec amuxruntime.ExecutionSpec) (amuxruntime.Process, error) {
	// Validate command
	if len(spec.Command) == 0 {
		return nil, amuxruntime.ErrInvalidCommand
	}

	// Get shell
	shell := getShell()

	// Create command without context to avoid automatic termination
	cmd := createCommand(ctx, spec, shell, false)

	// Setup command properties
	if err := setupCommand(cmd, spec); err != nil {
		return nil, err
	}

	// Configure process isolation for detached mode
	configureProcessIsolation(cmd, true)

	// Create process
	proc := createProcess(spec)
	proc.cmd = cmd

	// For detached processes, discard output to prevent blocking
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	proc.setState(amuxruntime.StateRunning)

	// Create metadata after process starts
	proc.metadata = &Metadata{
		PID:      cmd.Process.Pid,
		Detached: true,
	}

	// Try to get PGID (might not work on all platforms)
	if isProcessGroup(cmd) {
		proc.metadata.PGID = cmd.Process.Pid
	}

	// Store process
	r.processes.Store(proc.id, proc)

	// Monitor process completion
	go proc.monitor()

	// Detached processes don't handle context cancellation
	// They continue running even if the parent context is cancelled

	return proc, nil
}
