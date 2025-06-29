// Package runtime provides the core runtime abstraction layer for process execution.
package runtime

import (
	"context"
	"io"
	"time"
)

// Runtime represents an execution environment for processes
type Runtime interface {
	// Type returns the runtime type identifier (e.g., "local", "tmux", "docker")
	Type() string

	// Execute starts a new process in this runtime
	Execute(ctx context.Context, spec ExecutionSpec) (Process, error)

	// Find locates an existing process by ID
	Find(ctx context.Context, id string) (Process, error)

	// List returns all processes managed by this runtime
	List(ctx context.Context) ([]Process, error)

	// Validate checks if this runtime is properly configured and available
	Validate() error
}

// Process represents a running or completed process
type Process interface {
	// ID returns the unique identifier for this process
	ID() string

	// State returns the current state of the process
	State() ProcessState

	// Wait blocks until the process completes
	Wait(ctx context.Context) error

	// Stop gracefully stops the process
	Stop(ctx context.Context) error

	// Kill forcefully terminates the process
	Kill(ctx context.Context) error

	// Output returns readers for stdout and stderr
	Output() (stdout, stderr io.Reader)

	// ExitCode returns the exit code (valid after process completes)
	ExitCode() (int, error)

	// StartTime returns when the process was started
	StartTime() time.Time
}

// ExecutionSpec defines what to execute
type ExecutionSpec struct {
	Command     []string          // Command and arguments
	WorkingDir  string            // Working directory
	Environment map[string]string // Environment variables

	// Runtime-specific options
	Options RuntimeOptions
}

// RuntimeOptions is implemented by runtime-specific option types
//
//nolint:revive // RuntimeOptions is a clearer name than Options in this context
type RuntimeOptions interface {
	// IsRuntimeOptions is a marker method to ensure type safety
	IsRuntimeOptions()
}

// ProcessState represents the current state of a process
type ProcessState string

// Process state constants
const (
	StateStarting ProcessState = "starting" // Process is being initialized
	StateRunning  ProcessState = "running"  // Process is actively running
	StateStopped  ProcessState = "stopped"  // Process stopped normally
	StateFailed   ProcessState = "failed"   // Process failed or crashed
	StateUnknown  ProcessState = "unknown"  // Process state cannot be determined
)
