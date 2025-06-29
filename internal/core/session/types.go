package session

import (
	"context"
	"time"

	"github.com/aki/amux/internal/core/session/state"
	"github.com/google/uuid"
)

// ID is the full UUID of a session
type ID string

// Index is the short numeric identifier (1, 2, 3...)
type Index string

// Name is the human-readable name
type Name string

// Identifier can be any of: ID, Index, or Name
type Identifier string

// GenerateID generates a new unique session ID
func GenerateID() ID {
	return ID(uuid.New().String())
}

// String returns the string representation of the session ID
func (id ID) String() string {
	return string(id)
}

// Short returns the first 8 characters of the session ID
func (id ID) Short() string {
	if len(id) >= 8 {
		return string(id[:8])
	}
	return string(id)
}

// IsEmpty returns true if the ID is empty
func (id ID) IsEmpty() bool {
	return id == ""
}

// Type represents the type of session backend
type Type string

const (
	// TypeTmux indicates a tmux-based terminal session
	TypeTmux Type = "tmux"
	// Future: TypeClaudeCode, etc.
)

// Status is an alias for state.Status to maintain backward compatibility
// and avoid breaking changes in the public API
type Status = state.Status

// Re-export status constants for convenience
const (
	StatusCreated   = state.StatusCreated
	StatusStarting  = state.StatusStarting
	StatusRunning   = state.StatusRunning
	StatusStopping  = state.StatusStopping
	StatusCompleted = state.StatusCompleted
	StatusStopped   = state.StatusStopped
	StatusFailed    = state.StatusFailed
	StatusOrphaned  = state.StatusOrphaned
)

// ActivityTracking holds runtime metrics for activity monitoring
type ActivityTracking struct {
	LastOutputHash  uint32    `yaml:"lastOutputHash,omitempty"`
	LastOutputTime  time.Time `yaml:"lastOutputTime,omitempty"`
	LastStatusCheck time.Time `yaml:"lastStatusCheck,omitempty"`
}

// Options contains options for creating a new session
type Options struct {
	ID                  ID                // Optional: pre-generated session ID
	Type                Type              // Optional: session type (defaults to tmux)
	WorkspaceID         string            // Optional: workspace to run in
	AutoCreateWorkspace bool              // Create a new workspace if WorkspaceID is not provided
	AgentID             string            // Required: agent to run
	Command             string            // Optional: override agent command
	Environment         map[string]string // Optional: additional env vars
	InitialPrompt       string            // Optional: initial prompt to send after starting
	Name                string            // Optional: human-readable name for the session
	Description         string            // Optional: description of session purpose
	NoHooks             bool              // Skip hook execution
	RuntimeType         string            // Optional: override runtime type ("local", "tmux", etc.)
}

// Info contains metadata about a session
type Info struct {
	ID               string            `yaml:"id"`
	Index            string            `yaml:"-"` // Populated from ID mapper, not persisted
	Type             Type              `yaml:"type"`
	WorkspaceID      string            `yaml:"workspace_id"`
	AgentID          string            `yaml:"agent_id"`
	ActivityTracking ActivityTracking  `yaml:"activityTracking"`
	Command          string            `yaml:"command"`
	Environment      map[string]string `yaml:"environment,omitempty"`
	InitialPrompt    string            `yaml:"initial_prompt,omitempty"`
	PID              int               `yaml:"pid,omitempty"`
	TmuxSession      string            `yaml:"tmux_session,omitempty"`
	CreatedAt        time.Time         `yaml:"created_at"`
	StartedAt        *time.Time        `yaml:"started_at,omitempty"`
	StoppedAt        *time.Time        `yaml:"stopped_at,omitempty"`
	Error            string            `yaml:"error,omitempty"`
	StoragePath      string            `yaml:"storage_path,omitempty"`
	StateDir         string            `yaml:"state_dir,omitempty"`
	Name             string            `yaml:"name,omitempty"`
	Description      string            `yaml:"description,omitempty"`
	ShouldAutoAttach bool              `yaml:"-"`                      // Derived from agent config, not persisted
	RuntimeType      string            `yaml:"runtime_type,omitempty"` // Runtime type (local, tmux, etc.)
}

// Session represents an active or inactive agent session
type Session interface {
	// ID returns the unique session identifier
	ID() string

	// WorkspaceID returns the workspace this session runs in
	WorkspaceID() string

	// WorkspacePath returns the workspace path
	WorkspacePath() string

	// AgentID returns the agent running in this session
	AgentID() string

	// Type returns the session type
	Type() Type

	// Status returns the current session status
	Status() Status

	// Info returns the full session information
	Info() *Info

	// GetStoragePath returns the storage path for the session
	GetStoragePath() string

	// Start starts the session
	Start(ctx context.Context) error

	// Stop stops the session gracefully
	Stop(ctx context.Context) error
}

// TerminalSession represents a session with terminal capabilities
type TerminalSession interface {
	Session

	// Attach attaches to the session's terminal
	Attach() error

	// SendInput sends input to the session
	SendInput(ctx context.Context, input string) error

	// GetOutput returns recent output from the session
	GetOutput(maxLines int) ([]byte, error)

	// UpdateStatus updates the session status based on current output
	UpdateStatus(ctx context.Context) error
}
