package hooks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aki/amux/internal/cli/ui"
)

// Executor executes hooks
type Executor struct {
	configDir  string
	env        map[string]string
	dryRun     bool
	output     io.Writer
	workingDir string // New field for context-specific working directory
}

// NewExecutor creates a new hook executor
func NewExecutor(configDir string, env map[string]string) *Executor {
	return &Executor{
		configDir: configDir,
		env:       env,
		output:    os.Stdout,
	}
}

// WithDryRun sets dry run mode
func (e *Executor) WithDryRun(dryRun bool) *Executor {
	e.dryRun = dryRun
	return e
}

// WithOutput sets custom output writer
func (e *Executor) WithOutput(w io.Writer) *Executor {
	e.output = w
	return e
}

// WithWorkingDir sets the working directory for hook execution
func (e *Executor) WithWorkingDir(dir string) *Executor {
	e.workingDir = dir
	return e
}

// ExecuteHooks executes all hooks for the given event
func (e *Executor) ExecuteHooks(event Event, hooks []Hook) error {
	if len(hooks) == 0 {
		return nil
	}

	ui.Info("Running hooks for '%s'...", event)

	for i, hook := range hooks {
		result, err := e.executeHook(&hook, i+1, len(hooks))
		if err != nil {
			switch hook.OnError {
			case ErrorStrategyFail:
				ui.Error("Hook execution failed")
				return fmt.Errorf("hook '%s' failed: %w", hook.Name, err)
			case ErrorStrategyWarn:
				ui.Warning("Hook failed but continuing (on_error: warn)")
			case ErrorStrategyIgnore:
				// Silent continue
			}
		}

		if result != nil && result.ExitCode != 0 && hook.OnError == ErrorStrategyFail {
			return fmt.Errorf("hook '%s' exited with code %d", hook.Name, result.ExitCode)
		}
	}

	ui.Success("All hooks completed successfully")
	return nil
}

// executeHook executes a single hook
func (e *Executor) executeHook(hook *Hook, index, total int) (*ExecutionResult, error) {
	// Determine what to execute
	var cmdStr string
	if hook.Command != "" {
		cmdStr = hook.Command
	} else if hook.Script != "" {
		// For scripts, we need to check if the file exists
		scriptPath := hook.Script
		if !filepath.IsAbs(scriptPath) {
			// Relative to config directory
			scriptPath = filepath.Join(e.configDir, "..", scriptPath)
		}

		if _, err := os.Stat(scriptPath); err != nil {
			return nil, fmt.Errorf("script not found: %s", hook.Script)
		}

		cmdStr = scriptPath
	} else {
		return nil, fmt.Errorf("hook must have either 'command' or 'script'")
	}

	// Parse timeout
	timeout, err := time.ParseDuration(hook.Timeout)
	if err != nil {
		timeout = 5 * time.Minute // Default timeout
	}

	// Show progress
	ui.Info("  [%d/%d] %s", index, total, hook.Name)

	if e.dryRun {
		ui.Info("    > [DRY RUN] Would execute: %s", cmdStr)
		return &ExecutionResult{Hook: hook}, nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute command through shell to support redirections, pipes, etc.
	cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)

	// Set working directory
	if e.workingDir != "" {
		// Use context-specific working directory (workspace path)
		cmd.Dir = e.workingDir
	} else {
		// Fall back to project root (parent of .amux)
		cmd.Dir = filepath.Dir(e.configDir)
	}

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range e.env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range hook.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Capture output
	var outputBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(e.output, &outputBuf)
	cmd.Stderr = io.MultiWriter(e.output, &outputBuf)

	// Record start time
	result := &ExecutionResult{
		Hook:      hook,
		StartTime: time.Now(),
	}

	// Execute
	err = cmd.Run()
	result.EndTime = time.Now()
	result.Output = outputBuf.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = exitErr
		} else {
			result.Error = err
		}

		duration := result.EndTime.Sub(result.StartTime)
		ui.Error("  ✗ [%d/%d] %s - failed in %.1fs", index, total, hook.Name, duration.Seconds())
		return result, err
	}

	duration := result.EndTime.Sub(result.StartTime)
	ui.Success("  ✓ [%d/%d] %s - completed in %.1fs", index, total, hook.Name, duration.Seconds())

	return result, nil
}
