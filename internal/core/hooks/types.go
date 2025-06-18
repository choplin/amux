package hooks

import (
	"time"
)

// ErrorStrategy defines how to handle hook failures
type ErrorStrategy string

const (
	// ErrorStrategyFail stops execution on error
	ErrorStrategyFail ErrorStrategy = "fail"
	// ErrorStrategyWarn logs warning but continues
	ErrorStrategyWarn ErrorStrategy = "warn"
	// ErrorStrategyIgnore continues silently
	ErrorStrategyIgnore ErrorStrategy = "ignore"
)

// Hook represents a single hook configuration
type Hook struct {
	Name    string            `yaml:"name"`
	Command string            `yaml:"command,omitempty"`
	Script  string            `yaml:"script,omitempty"`
	Timeout string            `yaml:"timeout,omitempty"`
	OnError ErrorStrategy     `yaml:"on_error,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// Config represents the hooks configuration file
type Config struct {
	Hooks     map[string][]Hook `yaml:"hooks"`
	Templates map[string][]Hook `yaml:"templates,omitempty"`
}

// Event represents a hook event type
type Event string

const (
	// EventWorkspaceCreate fires after workspace creation
	EventWorkspaceCreate Event = "workspace_create"
	// EventWorkspaceRemove fires before workspace removal
	EventWorkspaceRemove Event = "workspace_remove"
	// EventSessionStart fires when session starts
	EventSessionStart Event = "session_start"
	// EventSessionStop fires when session stops
	EventSessionStop Event = "session_stop"
)

// ExecutionResult represents the result of hook execution
type ExecutionResult struct {
	Hook      *Hook
	StartTime time.Time
	EndTime   time.Time
	Output    string
	Error     error
	ExitCode  int
}

// TrustInfo represents hook trust information
type TrustInfo struct {
	Hash      string    `yaml:"hash"`
	TrustedAt time.Time `yaml:"trusted_at"`
	TrustedBy string    `yaml:"trusted_by"`
}
