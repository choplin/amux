// Package local provides a runtime implementation that executes processes directly using os/exec.
package local

import (
	"context"
	"fmt"
	"os/exec"

	amuxruntime "github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/runtime/proxy"
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

	// Create process record
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

	// Use foreground mode for local runtime
	args, err := proxy.BuildProxyCommandWithOptions(sessionID, command, spec.EnableLog, true)
	if err != nil {
		return nil, fmt.Errorf("failed to build proxy command: %w", err)
	}

	// Create the command
	var proxyCmd *exec.Cmd
	if ctx != nil {
		proxyCmd = exec.CommandContext(ctx, args[0], args[1:]...)
	} else {
		proxyCmd = exec.Command(args[0], args[1:]...)
	}

	// Setup command properties
	if err := setupCommand(proxyCmd, spec); err != nil {
		return nil, err
	}

	// Configure process isolation for foreground
	configureProcessIsolation(proxyCmd, false)

	proc.cmd = proxyCmd

	// Don't set stdin/stdout/stderr here - the proxy command will handle all I/O

	// Start the process
	if err := proxyCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	proc.setState(amuxruntime.StateRunning)

	// Create metadata after process starts
	proc.metadata = &Metadata{
		PID:      proxyCmd.Process.Pid,
		Detached: false,
	}

	// Try to get PGID (might not work on all platforms)
	if isProcessGroup(proxyCmd) {
		proc.metadata.PGID = proxyCmd.Process.Pid
	}

	// Store process
	r.processes.Store(proc.id, proc)

	// Store session mapping if session ID is provided
	if spec.SessionID != "" {
		r.sessions.Store(spec.SessionID, proc.id)
	}

	// Monitor process completion
	go proc.monitor()

	// For foreground execution, wait for the process to complete
	waitErr := proc.Wait(ctx)
	if waitErr != nil {
		return nil, waitErr
	}

	return proc, nil
}
