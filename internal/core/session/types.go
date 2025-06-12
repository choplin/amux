package session

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ID represents a unique session identifier
type ID string

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

// Status represents the current state of a session
type Status string

// String returns the string representation of the status
func (s Status) String() string {
	return string(s)
}

// IsRunning returns true if the session is in a running state (working or idle)
func (s Status) IsRunning() bool {
	return s == StatusWorking || s == StatusIdle
}

// IsTerminal returns true if the session is in a terminal state (stopped or failed)
func (s Status) IsTerminal() bool {
	return s == StatusStopped || s == StatusFailed
}

const (
	// StatusCreated indicates a session has been created but not started
	StatusCreated Status = "created"
	// StatusWorking indicates a session is actively processing (output changing)
	StatusWorking Status = "working"
	// StatusIdle indicates a session is waiting for input (no recent output)
	StatusIdle Status = "idle"
	// StatusStopped indicates a session has been stopped normally
	StatusStopped Status = "stopped"
	// StatusFailed indicates a session has failed or crashed
	StatusFailed Status = "failed"
)

// Options contains options for creating a new session
type Options struct {
	ID            ID                // Optional: pre-generated session ID
	WorkspaceID   string            // Required: workspace to run in
	AgentID       string            // Required: agent to run
	Command       string            // Optional: override agent command
	Environment   map[string]string // Optional: additional env vars
	InitialPrompt string            // Optional: initial prompt to send after starting
}

// Info contains metadata about a session
type Info struct {
	ID              string            `yaml:"id"`
	Index           string            `yaml:"-"` // Populated from ID mapper, not persisted
	WorkspaceID     string            `yaml:"workspace_id"`
	AgentID         string            `yaml:"agent_id"`
	Status          Status            `yaml:"status"`
	Command         string            `yaml:"command"`
	Environment     map[string]string `yaml:"environment,omitempty"`
	InitialPrompt   string            `yaml:"initial_prompt,omitempty"`
	PID             int               `yaml:"pid,omitempty"`
	TmuxSession     string            `yaml:"tmux_session,omitempty"`
	CreatedAt       time.Time         `yaml:"created_at"`
	StartedAt       *time.Time        `yaml:"started_at,omitempty"`
	StoppedAt       *time.Time        `yaml:"stopped_at,omitempty"`
	StatusChangedAt time.Time         `yaml:"-"` // When status last changed, not persisted
	Error           string            `yaml:"error,omitempty"`
}

// Session represents an active or inactive agent session
type Session interface {
	// ID returns the unique session identifier
	ID() string

	// WorkspaceID returns the workspace this session runs in
	WorkspaceID() string

	// AgentID returns the agent running in this session
	AgentID() string

	// Status returns the current session status
	Status() Status

	// Info returns the full session information
	Info() *Info

	// Start starts the session
	Start(ctx context.Context) error

	// Stop stops the session gracefully
	Stop() error

	// Attach attaches to the session's terminal
	Attach() error

	// SendInput sends input to the session
	SendInput(input string) error

	// GetOutput returns recent output from the session
	GetOutput(maxLines int) ([]byte, error)
}

// Store persists session metadata
type Store interface {
	// Save saves session info
	Save(info *Info) error

	// Load loads session info by ID
	Load(id string) (*Info, error)

	// List lists all session infos
	List() ([]*Info, error)

	// Delete deletes session info
	Delete(id string) error
}
