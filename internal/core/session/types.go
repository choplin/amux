package session

import (
	"context"
	"time"

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
	// TypeBlocking indicates a blocking command execution session
	TypeBlocking Type = "blocking"
	// Future: TypeClaudeCode, TypeCommand, etc.
)

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

// IsTerminal returns true if the session is in a terminal state (completed, stopped, failed or orphaned)
func (s Status) IsTerminal() bool {
	return s == StatusCompleted || s == StatusStopped || s == StatusFailed || s == StatusOrphaned
}

const (
	// StatusCreated indicates a session has been created but not started
	StatusCreated Status = "created"
	// StatusWorking indicates a session is actively processing (output changing)
	StatusWorking Status = "working"
	// StatusIdle indicates a session is waiting for input (no recent output)
	StatusIdle Status = "idle"
	// StatusCompleted indicates a session command has finished successfully
	StatusCompleted Status = "completed"
	// StatusStopped indicates a session has been stopped normally
	StatusStopped Status = "stopped"
	// StatusFailed indicates a session has failed or crashed
	StatusFailed Status = "failed"
	// StatusOrphaned indicates a session with missing dependencies (e.g., deleted workspace)
	StatusOrphaned Status = "orphaned"
)

// StatusState holds runtime state for status tracking
type StatusState struct {
	Status          Status    `yaml:"status"`
	StatusChangedAt time.Time `yaml:"statusChangedAt"`
	LastOutputHash  uint32    `yaml:"lastOutputHash,omitempty"`
	LastOutputTime  time.Time `yaml:"lastOutputTime,omitempty"`
	LastStatusCheck time.Time `yaml:"lastStatusCheck,omitempty"`
}

// OutputMode represents how session output is captured
type OutputMode string

const (
	// OutputModeBuffer captures output in memory with size limit
	OutputModeBuffer OutputMode = "buffer"
	// OutputModeFile captures output to a file with no size limit
	OutputModeFile OutputMode = "file"
	// OutputModeCircular captures output in a circular buffer keeping only recent data
	OutputModeCircular OutputMode = "circular"
)

// OutputConfig configures how session output is captured
type OutputConfig struct {
	Mode       OutputMode `yaml:"mode"`
	BufferSize int64      `yaml:"bufferSize,omitempty"` // For buffer/circular modes (bytes)
	FilePath   string     `yaml:"filePath,omitempty"`   // For file mode (auto-generated if empty)
}

// GetDefaultOutputConfig returns the default output configuration
func GetDefaultOutputConfig() *OutputConfig {
	return &OutputConfig{
		Mode:       OutputModeBuffer,
		BufferSize: 10 * 1024 * 1024, // 10MB
	}
}

// Options contains options for creating a new session
type Options struct {
	ID            ID                // Optional: pre-generated session ID
	Type          Type              // Optional: session type (defaults to tmux)
	WorkspaceID   string            // Required: workspace to run in
	AgentID       string            // Required: agent to run
	Command       string            // Optional: override agent command
	Environment   map[string]string // Optional: additional env vars
	InitialPrompt string            // Optional: initial prompt to send after starting
	Name          string            // Optional: human-readable name for the session
	Description   string            // Optional: description of session purpose

	// Blocking session specific options
	BlockingCommand string        // Command for blocking sessions
	BlockingArgs    []string      // Arguments for blocking sessions
	OutputConfig    *OutputConfig // Optional output configuration
}

// Info contains metadata about a session
type Info struct {
	ID            string            `yaml:"id"`
	Index         string            `yaml:"-"` // Populated from ID mapper, not persisted
	Type          Type              `yaml:"type"`
	WorkspaceID   string            `yaml:"workspace_id"`
	AgentID       string            `yaml:"agent_id"`
	StatusState   StatusState       `yaml:"statusState"`
	Command       string            `yaml:"command"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	InitialPrompt string            `yaml:"initial_prompt,omitempty"`
	PID           int               `yaml:"pid,omitempty"`
	TmuxSession   string            `yaml:"tmux_session,omitempty"`
	CreatedAt     time.Time         `yaml:"created_at"`
	StartedAt     *time.Time        `yaml:"started_at,omitempty"`
	StoppedAt     *time.Time        `yaml:"stopped_at,omitempty"`
	Error         string            `yaml:"error,omitempty"`
	StoragePath   string            `yaml:"storage_path,omitempty"`
	Name          string            `yaml:"name,omitempty"`
	Description   string            `yaml:"description,omitempty"`

	// Blocking session specific fields
	BlockingCommand string        `yaml:"blocking_command,omitempty"`
	BlockingArgs    []string      `yaml:"blocking_args,omitempty"`
	OutputConfig    *OutputConfig `yaml:"output_config,omitempty"`
	ExitCode        int           `yaml:"exit_code,omitempty"`
	BufferFull      bool          `yaml:"buffer_full,omitempty"`
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
	SendInput(input string) error

	// GetOutput returns recent output from the session
	GetOutput(maxLines int) ([]byte, error)

	// UpdateStatus updates the session status based on current output
	UpdateStatus(ctx context.Context) error
}
