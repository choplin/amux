package task

import (
	"fmt"
	"strings"
	"time"
)

// Validator provides task validation functionality
type Validator struct{}

// NewValidator creates a new task validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateTask performs comprehensive validation on a task
func (v *Validator) ValidateTask(task *Task) error {
	if err := v.validateBasicFields(task); err != nil {
		return err
	}

	if err := v.validateLifecycle(task); err != nil {
		return err
	}

	if err := v.validateTimeout(task); err != nil {
		return err
	}

	if err := v.validateEnvironment(task); err != nil {
		return err
	}

	return nil
}

// validateBasicFields validates required fields
func (v *Validator) validateBasicFields(task *Task) error {
	if task.Name == "" {
		return fmt.Errorf("task name cannot be empty")
	}

	if strings.TrimSpace(task.Name) != task.Name {
		return fmt.Errorf("task name cannot have leading or trailing whitespace")
	}

	if task.Command == "" {
		return fmt.Errorf("task command cannot be empty")
	}

	return nil
}

// validateLifecycle validates lifecycle settings
func (v *Validator) validateLifecycle(task *Task) error {
	if task.Lifecycle == "" {
		// Default to oneshot
		task.Lifecycle = LifecycleOneshot
		return nil
	}

	return ValidateLifecycleType(string(task.Lifecycle))
}

// validateTimeout validates timeout settings
func (v *Validator) validateTimeout(task *Task) error {
	if task.Timeout == "" {
		return nil
	}

	// Timeout only makes sense for oneshot tasks
	if task.Lifecycle == LifecycleDaemon {
		return fmt.Errorf("timeout cannot be specified for daemon tasks")
	}

	// Validate timeout format
	_, err := time.ParseDuration(task.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout format: %w", err)
	}

	return nil
}

// validateEnvironment validates environment variables
func (v *Validator) validateEnvironment(task *Task) error {
	for key := range task.Env {
		if key == "" {
			return fmt.Errorf("environment variable name cannot be empty")
		}
		if strings.Contains(key, "=") {
			return fmt.Errorf("environment variable name cannot contain '=': %s", key)
		}
	}
	return nil
}

// ValidateTaskList validates a list of tasks and their dependencies
func (v *Validator) ValidateTaskList(tasks []*Task) error {
	// Create a map for quick lookup
	taskMap := make(map[string]*Task)

	// First pass: validate individual tasks and check for duplicates
	for _, task := range tasks {
		if err := v.ValidateTask(task); err != nil {
			return fmt.Errorf("invalid task %s: %w", task.Name, err)
		}

		if _, exists := taskMap[task.Name]; exists {
			return fmt.Errorf("duplicate task name: %s", task.Name)
		}

		taskMap[task.Name] = task
	}

	// Second pass: validate dependencies
	for _, task := range tasks {
		if err := v.validateDependencies(task, taskMap); err != nil {
			return fmt.Errorf("task %s: %w", task.Name, err)
		}
	}

	// Check for circular dependencies
	for _, task := range tasks {
		if err := v.checkCircularDependencies(task.Name, taskMap, make(map[string]bool)); err != nil {
			return err
		}
	}

	return nil
}

// validateDependencies validates that all dependencies exist
func (v *Validator) validateDependencies(task *Task, taskMap map[string]*Task) error {
	for _, dep := range task.DependsOn {
		if _, exists := taskMap[dep]; !exists {
			return fmt.Errorf("depends on non-existent task: %s", dep)
		}
	}
	return nil
}

// checkCircularDependencies checks for circular dependencies
func (v *Validator) checkCircularDependencies(taskName string, taskMap map[string]*Task, visited map[string]bool) error {
	if visited[taskName] {
		return fmt.Errorf("circular dependency detected involving task: %s", taskName)
	}

	visited[taskName] = true
	defer delete(visited, taskName)

	task, exists := taskMap[taskName]
	if !exists {
		return fmt.Errorf("task not found: %s", taskName)
	}

	for _, dep := range task.DependsOn {
		if err := v.checkCircularDependencies(dep, taskMap, visited); err != nil {
			return err
		}
	}

	return nil
}
