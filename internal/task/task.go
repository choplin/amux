package task

import (
	"fmt"
	"strings"
)

// LifecycleType represents how a task should be executed
type LifecycleType string

const (
	// LifecycleOneshot represents tasks that run once and exit
	LifecycleOneshot LifecycleType = "oneshot"
	// LifecycleDaemon represents tasks that run continuously
	LifecycleDaemon LifecycleType = "daemon"
)

// ValidateLifecycleType checks if the lifecycle type is valid
func ValidateLifecycleType(lifecycle string) error {
	switch LifecycleType(strings.ToLower(lifecycle)) {
	case LifecycleOneshot, LifecycleDaemon:
		return nil
	default:
		return fmt.Errorf("invalid lifecycle type: %s (must be 'oneshot' or 'daemon')", lifecycle)
	}
}

// Task represents a reusable command template
type Task struct {
	// Name is the unique identifier for the task
	Name string `yaml:"name"`

	// Description provides a human-readable explanation of what the task does
	Description string `yaml:"description,omitempty"`

	// Command is the command template to execute
	Command string `yaml:"command"`

	// Lifecycle defines how the task should be executed
	Lifecycle LifecycleType `yaml:"lifecycle,omitempty"`

	// WorkingDir specifies the working directory for the task
	WorkingDir string `yaml:"working_dir,omitempty"`

	// Env contains additional environment variables for the task
	Env map[string]string `yaml:"env,omitempty"`

	// DependsOn lists task names that must be running before this task
	DependsOn []string `yaml:"depends_on,omitempty"`

	// Timeout specifies the maximum duration for the task (only for oneshot)
	Timeout string `yaml:"timeout,omitempty"`
}

// Validate checks if the task definition is valid
func (t *Task) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("task name cannot be empty")
	}

	if t.Command == "" {
		return fmt.Errorf("task command cannot be empty")
	}

	// Set default lifecycle if not specified
	if t.Lifecycle == "" {
		t.Lifecycle = LifecycleOneshot
	}

	// Validate lifecycle type
	if err := ValidateLifecycleType(string(t.Lifecycle)); err != nil {
		return err
	}

	// Validate timeout only applies to oneshot tasks
	if t.Timeout != "" && t.Lifecycle != LifecycleOneshot {
		return fmt.Errorf("timeout can only be specified for oneshot tasks")
	}

	return nil
}

// IsOneshot returns true if the task is a oneshot task
func (t *Task) IsOneshot() bool {
	return t.Lifecycle == LifecycleOneshot || t.Lifecycle == ""
}

// IsDaemon returns true if the task is a daemon task
func (t *Task) IsDaemon() bool {
	return t.Lifecycle == LifecycleDaemon
}
