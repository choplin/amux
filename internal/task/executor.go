// Package task provides task management and execution functionality for amux.
package task

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Executor handles task execution
type Executor struct {
	manager *Manager
}

// NewExecutor creates a new task executor
func NewExecutor(manager *Manager) *Executor {
	return &Executor{
		manager: manager,
	}
}

// ExecuteTask executes a task in the given working directory
// Note: This is a simplified implementation for Phase 2.
// Full integration with Runtime and Session will be done in Phase 3.
func (e *Executor) ExecuteTask(ctx context.Context, taskName string, workingDir string) error {
	task, err := e.manager.GetTask(taskName)
	if err != nil {
		return err
	}

	// Validate dependencies
	if err := e.ValidateDependencies(taskName); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	// Set working directory
	workDir := task.WorkingDir
	if workDir == "" {
		workDir = workingDir
	}

	// Execute based on lifecycle
	switch task.Lifecycle {
	case LifecycleOneshot:
		return e.executeOneshot(ctx, task, workDir)
	case LifecycleDaemon:
		return e.executeDaemon(ctx, task, workDir)
	default:
		// Default to oneshot
		return e.executeOneshot(ctx, task, workDir)
	}
}

// executeOneshot executes a oneshot task
func (e *Executor) executeOneshot(ctx context.Context, task *Task, workDir string) error {
	// Parse timeout if specified
	var timeoutDuration time.Duration
	if task.Timeout != "" {
		d, err := time.ParseDuration(task.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout: %w", err)
		}
		timeoutDuration = d
	}

	// Create context with timeout if specified
	execCtx := ctx
	if timeoutDuration > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, timeoutDuration)
		defer cancel()
	}

	// Execute command
	cmd := exec.CommandContext(execCtx, "sh", "-c", task.Command)
	cmd.Dir = workDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range task.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("task timed out after %s", task.Timeout)
		}
		return fmt.Errorf("task execution failed: %w", err)
	}

	return nil
}

// executeDaemon executes a daemon task
func (e *Executor) executeDaemon(ctx context.Context, task *Task, workDir string) error {
	// For now, daemon tasks are executed the same way as oneshot tasks
	// In Phase 3, this will integrate with session management
	cmd := exec.CommandContext(ctx, "sh", "-c", task.Command)
	cmd.Dir = workDir

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range task.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon task: %w", err)
	}

	// For now, we just start the daemon and return
	// In Phase 3, this will be managed by the session system
	fmt.Printf("Daemon task started with PID: %d\n", cmd.Process.Pid)

	return nil
}

// ValidateDependencies validates that all task dependencies are satisfied
func (e *Executor) ValidateDependencies(taskName string) error {
	task, err := e.manager.GetTask(taskName)
	if err != nil {
		return err
	}

	// Check each dependency
	for _, dep := range task.DependsOn {
		if !e.manager.HasTask(dep) {
			return fmt.Errorf("dependency not found: %s", dep)
		}

		// Recursively validate dependencies
		if err := e.ValidateDependencies(dep); err != nil {
			return fmt.Errorf("dependency %s has invalid dependencies: %w", dep, err)
		}
	}

	return nil
}
