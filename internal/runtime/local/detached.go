package local

import (
	"context"
	"fmt"
	"os/exec"

	amuxruntime "github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/runtime/proxy"
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

	// Create process first to get ID
	proc := createProcess(spec)

	// Build proxy command arguments
	sessionID := spec.SessionID
	if sessionID == "" {
		sessionID = proc.id
	}

	// Build command based on spec
	var command []string
	if len(spec.Command) == 1 {
		// Single command, run through shell
		shell := proxy.GetShell()
		command = []string{shell, "-c", spec.Command[0]}
	} else {
		// Multiple arguments, execute directly
		command = spec.Command
	}

	args, err := proxy.BuildProxyCommand(sessionID, command, spec.EnableLog)
	if err != nil {
		return nil, fmt.Errorf("failed to build proxy command: %w", err)
	}

	// Create the command (no context for detached processes)
	cmd := exec.Command(args[0], args[1:]...)

	// Setup command properties
	if err := setupCommand(cmd, spec); err != nil {
		return nil, err
	}

	// Configure process isolation for detached mode
	configureProcessIsolation(cmd, true)

	// Set command on process
	proc.cmd = cmd

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

	// Store session mapping if session ID is provided
	if spec.SessionID != "" {
		r.sessions.Store(spec.SessionID, proc.id)
	}

	// Monitor process completion
	go func() {
		proc.monitor()
	}()

	// Detached processes don't handle context cancellation
	// They continue running even if the parent context is cancelled

	return proc, nil
}
