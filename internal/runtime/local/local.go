// Package local provides a runtime implementation that executes processes directly using os/exec.
package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"

	amuxruntime "github.com/aki/amux/internal/runtime"
)

// Runtime implements the local process runtime using os/exec
type Runtime struct {
	processes sync.Map // map[string]*Process
}

// New creates a new local runtime
func New() *Runtime {
	return &Runtime{}
}

// Type returns the runtime type identifier
func (r *Runtime) Type() string {
	return "local"
}

// Execute starts a new process in the local runtime
func (r *Runtime) Execute(ctx context.Context, spec amuxruntime.ExecutionSpec) (amuxruntime.Process, error) {
	// Validate command
	if len(spec.Command) == 0 {
		return nil, amuxruntime.ErrInvalidCommand
	}

	// Get options with defaults
	opts, _ := spec.Options.(Options)
	if opts.Shell == "" {
		opts.Shell = os.Getenv("SHELL")
		if opts.Shell == "" {
			if runtime.GOOS == "windows" {
				opts.Shell = "cmd"
			} else {
				opts.Shell = "/bin/sh"
			}
		}
	}

	// Create command
	var cmd *exec.Cmd
	if opts.Detach {
		// For detached processes, don't use CommandContext to avoid automatic termination
		if len(spec.Command) == 1 {
			// Single command, run through shell
			if runtime.GOOS == "windows" {
				cmd = exec.Command(opts.Shell, "/c", spec.Command[0])
			} else {
				cmd = exec.Command(opts.Shell, "-c", spec.Command[0])
			}
		} else {
			// Multiple arguments, run directly
			cmd = exec.Command(spec.Command[0], spec.Command[1:]...)
		}
	} else {
		// For foreground processes, use CommandContext for proper cancellation
		if len(spec.Command) == 1 {
			// Single command, run through shell
			if runtime.GOOS == "windows" {
				cmd = exec.CommandContext(ctx, opts.Shell, "/c", spec.Command[0])
			} else {
				cmd = exec.CommandContext(ctx, opts.Shell, "-c", spec.Command[0])
			}
		} else {
			// Multiple arguments, run directly
			cmd = exec.CommandContext(ctx, spec.Command[0], spec.Command[1:]...)
		}
	}

	// Set working directory
	if spec.WorkingDir != "" {
		if _, err := os.Stat(spec.WorkingDir); err != nil {
			return nil, fmt.Errorf("working directory does not exist: %w", err)
		}
		cmd.Dir = spec.WorkingDir
	}

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range spec.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Configure process isolation
	configureProcessIsolation(cmd, opts.Detach)

	// Create process
	proc := &Process{
		id:        uuid.New().String(),
		cmd:       cmd,
		spec:      spec,
		state:     amuxruntime.StateStarting,
		startTime: time.Now(),
		opts:      opts,
		done:      make(chan struct{}),
	}

	// Set up output handling
	if !opts.Detach {
		// For foreground processes, inherit stdout/stderr for real-time output
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// For detached processes, discard output to prevent blocking
		cmd.Stdout = nil
		cmd.Stderr = nil
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	proc.setState(amuxruntime.StateRunning)

	// Create metadata after process starts
	proc.metadata = &Metadata{
		PID:      cmd.Process.Pid,
		Detached: opts.Detach,
	}

	// Try to get PGID (might not work on all platforms)
	if cmd.SysProcAttr != nil && cmd.SysProcAttr.Setpgid {
		proc.metadata.PGID = cmd.Process.Pid
	}

	// Store process
	r.processes.Store(proc.id, proc)

	// Monitor process completion
	go proc.monitor()

	// Handle context cancellation only for foreground processes
	if !opts.Detach {
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
	}

	// For foreground processes, wait for completion
	if !opts.Detach {
		go func() {
			// Wait for the process to complete
			<-proc.done
		}()
	}

	return proc, nil
}

// Find locates an existing process by ID
func (r *Runtime) Find(ctx context.Context, id string) (amuxruntime.Process, error) {
	if proc, ok := r.processes.Load(id); ok {
		return proc.(*Process), nil
	}
	return nil, amuxruntime.ErrProcessNotFound
}

// List returns all processes managed by this runtime
func (r *Runtime) List(ctx context.Context) ([]amuxruntime.Process, error) {
	var processes []amuxruntime.Process
	r.processes.Range(func(key, value interface{}) bool {
		processes = append(processes, value.(*Process))
		return true
	})
	return processes, nil
}

// Validate checks if this runtime is properly configured and available
func (r *Runtime) Validate() error {
	// Local runtime is always available
	return nil
}

// Process represents a local process
type Process struct {
	id        string
	cmd       *exec.Cmd
	spec      amuxruntime.ExecutionSpec
	state     amuxruntime.ProcessState
	startTime time.Time
	opts      Options
	mu        sync.RWMutex
	done      chan struct{}
	doneOnce  sync.Once
	metadata  *Metadata
}

// ID returns the unique identifier for this process
func (p *Process) ID() string {
	return p.id
}

// State returns the current state of the process
func (p *Process) State() amuxruntime.ProcessState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// Wait blocks until the process completes
func (p *Process) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		// Process already completed, check exit status
		if p.cmd.ProcessState != nil {
			if p.cmd.ProcessState.Success() {
				return nil
			}
			// Return the exit error
			return &exec.ExitError{ProcessState: p.cmd.ProcessState}
		}
		return nil
	}
}

// Stop gracefully stops the process (SIGTERM)
func (p *Process) Stop(ctx context.Context) error {
	p.mu.Lock()
	if p.state != amuxruntime.StateRunning {
		p.mu.Unlock()
		return amuxruntime.ErrProcessAlreadyDone
	}
	p.mu.Unlock()

	// For process groups, send signal to the entire group
	if p.cmd.SysProcAttr != nil && p.cmd.SysProcAttr.Setpgid {
		// Send signal to process group (negative PID)
		if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM to process group: %w", err)
		}
	} else {
		if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}
	}

	// Give process time to stop gracefully
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		// Force kill if still running
		return p.Kill(ctx)
	case <-p.done:
		return nil
	}
}

// Kill forcefully terminates the process (SIGKILL)
func (p *Process) Kill(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state != amuxruntime.StateRunning {
		return amuxruntime.ErrProcessAlreadyDone
	}

	// For process groups, kill the entire group
	if p.cmd.SysProcAttr != nil && p.cmd.SysProcAttr.Setpgid {
		// Kill process group (negative PID)
		if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL); err != nil {
			return fmt.Errorf("failed to kill process group: %w", err)
		}
	} else {
		if err := p.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	return nil
}

// Output returns readers for stdout and stderr
func (p *Process) Output() (stdout, stderr io.Reader) {
	// Output is not captured in the current implementation
	// For foreground processes, output goes directly to terminal
	// For detached processes, output is discarded
	return nil, nil
}

// ExitCode returns the exit code (valid after process completes)
func (p *Process) ExitCode() (int, error) {
	select {
	case <-p.done:
		if p.cmd.ProcessState != nil {
			return p.cmd.ProcessState.ExitCode(), nil
		}
		return -1, fmt.Errorf("process state not available")
	default:
		return -1, fmt.Errorf("process still running")
	}
}

// StartTime returns when the process was started
func (p *Process) StartTime() time.Time {
	return p.startTime
}

// setState updates the process state
func (p *Process) setState(state amuxruntime.ProcessState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
}

// monitor waits for the process to complete and updates its state
func (p *Process) monitor() {
	err := p.cmd.Wait()

	p.mu.Lock()
	if err != nil {
		p.state = amuxruntime.StateFailed
	} else {
		p.state = amuxruntime.StateStopped
	}
	p.mu.Unlock()

	// Signal completion
	p.doneOnce.Do(func() {
		close(p.done)
	})
}

// Metadata returns runtime-specific metadata
func (p *Process) Metadata() amuxruntime.Metadata {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.metadata == nil {
		return nil
	}

	// Return a copy to prevent modification
	return &Metadata{
		PID:      p.metadata.PID,
		PGID:     p.metadata.PGID,
		Detached: p.metadata.Detached,
	}
}
