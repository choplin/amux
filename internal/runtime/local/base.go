// Package local provides runtime implementations that execute processes directly using os/exec.
package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"

	amuxruntime "github.com/aki/amux/internal/runtime"
)

// baseRuntime provides common functionality for local runtime implementations
type baseRuntime struct {
	processes sync.Map // map[string]*Process
	sessions  sync.Map // map[sessionID]processID
}

// Find locates an existing process by ID
func (r *baseRuntime) Find(ctx context.Context, id string) (amuxruntime.Process, error) {
	if proc, ok := r.processes.Load(id); ok {
		return proc.(*Process), nil
	}
	return nil, amuxruntime.ErrProcessNotFound
}

// List returns all processes managed by this runtime
func (r *baseRuntime) List(ctx context.Context) ([]amuxruntime.Process, error) {
	var processes []amuxruntime.Process
	r.processes.Range(func(key, value interface{}) bool {
		processes = append(processes, value.(*Process))
		return true
	})
	return processes, nil
}

// Validate checks if this runtime is properly configured and available
func (r *baseRuntime) Validate() error {
	// Local runtime is always available
	return nil
}

// Stop gracefully stops a session
func (r *baseRuntime) Stop(ctx context.Context, sessionID string) error {
	processID, ok := r.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	proc, ok := r.processes.Load(processID)
	if !ok {
		return fmt.Errorf("process not found for session: %s", sessionID)
	}

	return proc.(*Process).Stop(ctx)
}

// Kill forcefully terminates a session
func (r *baseRuntime) Kill(ctx context.Context, sessionID string) error {
	processID, ok := r.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	proc, ok := r.processes.Load(processID)
	if !ok {
		return fmt.Errorf("process not found for session: %s", sessionID)
	}

	return proc.(*Process).Kill(ctx)
}

// SendInput sends input to a session
func (r *baseRuntime) SendInput(ctx context.Context, sessionID string, input string) error {
	processID, ok := r.sessions.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	proc, ok := r.processes.Load(processID)
	if !ok {
		return fmt.Errorf("process not found for session: %s", sessionID)
	}

	return proc.(*Process).SendInput(input)
}

// setupCommand configures common command properties
func setupCommand(cmd *exec.Cmd, spec amuxruntime.ExecutionSpec) error {
	// Set working directory
	if spec.WorkingDir != "" {
		if _, err := os.Stat(spec.WorkingDir); err != nil {
			return fmt.Errorf("working directory does not exist: %w", err)
		}
		cmd.Dir = spec.WorkingDir
	}

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range spec.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	return nil
}

// Process represents a local process
type Process struct {
	id        string
	cmd       *exec.Cmd
	spec      amuxruntime.ExecutionSpec
	state     amuxruntime.ProcessState
	startTime time.Time
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

	// Send stop signal to process or process group
	if err := signalStop(p.cmd); err != nil {
		return err
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

	// Kill process or process group
	if err := signalKill(p.cmd); err != nil {
		return err
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

// SendInput sends input to the process
func (p *Process) SendInput(input string) error {
	// Local processes don't support input sending through this interface
	// Input would be handled by the proxy process if needed
	return fmt.Errorf("input sending not supported for local processes")
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

// createProcess creates a new Process instance
func createProcess(spec amuxruntime.ExecutionSpec) *Process {
	return &Process{
		id:        uuid.New().String(),
		spec:      spec,
		state:     amuxruntime.StateStarting,
		startTime: time.Now(),
		done:      make(chan struct{}),
	}
}
