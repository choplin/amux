package session

import (
	"context"
	"time"
)

// SessionStatus represents the current state of a session
type SessionStatus string

const (
	StatusCreated SessionStatus = "created"
	StatusRunning SessionStatus = "running"
	StatusStopped SessionStatus = "stopped"
	StatusFailed  SessionStatus = "failed"
)

// SessionOptions contains options for creating a new session
type SessionOptions struct {
	WorkspaceID string            // Required: workspace to run in
	AgentID     string            // Required: agent to run
	Command     string            // Optional: override agent command
	Environment map[string]string // Optional: additional env vars
}

// SessionInfo contains metadata about a session
type SessionInfo struct {
	ID          string            `yaml:"id"`
	ShortID     string            `yaml:"-"` // Populated from ID mapper, not persisted
	WorkspaceID string            `yaml:"workspace_id"`
	AgentID     string            `yaml:"agent_id"`
	Status      SessionStatus     `yaml:"status"`
	Command     string            `yaml:"command"`
	Environment map[string]string `yaml:"environment,omitempty"`
	PID         int               `yaml:"pid,omitempty"`
	TmuxSession string            `yaml:"tmux_session,omitempty"`
	CreatedAt   time.Time         `yaml:"created_at"`
	StartedAt   *time.Time        `yaml:"started_at,omitempty"`
	StoppedAt   *time.Time        `yaml:"stopped_at,omitempty"`
	Error       string            `yaml:"error,omitempty"`
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
	Status() SessionStatus

	// Info returns the full session information
	Info() *SessionInfo

	// Start starts the session
	Start(ctx context.Context) error

	// Stop stops the session gracefully
	Stop() error

	// Attach attaches to the session's terminal
	Attach() error

	// SendInput sends input to the session
	SendInput(input string) error

	// GetOutput returns recent output from the session
	GetOutput() ([]byte, error)
}

// SessionManager manages the lifecycle of agent sessions
type SessionManager interface {
	// CreateSession creates a new session
	CreateSession(opts SessionOptions) (Session, error)

	// GetSession retrieves a session by ID
	GetSession(id string) (Session, error)

	// ListSessions lists all sessions
	ListSessions() ([]Session, error)

	// RemoveSession removes a stopped session
	RemoveSession(id string) error

	// CleanupOrphaned cleans up orphaned sessions
	CleanupOrphaned() error
}

// SessionStore persists session metadata
type SessionStore interface {
	// Save saves session info
	Save(info *SessionInfo) error

	// Load loads session info by ID
	Load(id string) (*SessionInfo, error)

	// List lists all session infos
	List() ([]*SessionInfo, error)

	// Delete deletes session info
	Delete(id string) error
}
