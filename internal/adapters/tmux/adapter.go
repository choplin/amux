// Package tmux provides a tmux adapter for terminal multiplexing.
package tmux

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RealAdapter provides real tmux operations
type RealAdapter struct {
	tmuxPath string
}

// NewAdapter creates a new tmux adapter
func NewAdapter() (Adapter, error) {
	// Check if tmux is available
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux not found: %w", err)
	}

	return &RealAdapter{
		tmuxPath: tmuxPath,
	}, nil
}

// IsAvailable checks if tmux is available on the system
func (a *RealAdapter) IsAvailable() bool {
	cmd := exec.Command(a.tmuxPath, "-V")
	return cmd.Run() == nil
}

// CreateSession creates a new tmux session
func (a *RealAdapter) CreateSession(sessionName, workDir string) error {
	return a.CreateSessionWithOptions(CreateSessionOptions{
		SessionName: sessionName,
		WorkDir:     workDir,
	})
}

// CreateSessionWithOptions creates a new tmux session with custom options
func (a *RealAdapter) CreateSessionWithOptions(opts CreateSessionOptions) error {
	// Build command arguments
	args := []string{"new-session", "-d", "-s", opts.SessionName, "-c", opts.WorkDir}

	// Add window name if specified
	if opts.WindowName != "" {
		args = append(args, "-n", opts.WindowName)
	}

	// Add environment variables using -e option
	for key, value := range opts.Environment {
		args = append(args, "-e", key+"="+value)
	}

	// Add shell as initial command if specified
	if opts.Shell != "" {
		args = append(args, opts.Shell)
	}

	// Create new session in detached mode with working directory
	// Set TERM to avoid terminal issues
	cmd := exec.Command(a.tmuxPath, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	return nil
}

// SessionExists checks if a tmux session exists
func (a *RealAdapter) SessionExists(sessionName string) bool {
	cmd := exec.Command(a.tmuxPath, "has-session", "-t", sessionName)
	return cmd.Run() == nil
}

// KillSession kills a tmux session
func (a *RealAdapter) KillSession(sessionName string) error {
	if !a.SessionExists(sessionName) {
		return nil // Already gone
	}

	cmd := exec.Command(a.tmuxPath, "kill-session", "-t", sessionName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to kill tmux session: %w", err)
	}
	return nil
}

// SendKeys sends keystrokes to a tmux session
func (a *RealAdapter) SendKeys(sessionName, keys string) error {
	// Check if session exists first
	if !a.SessionExists(sessionName) {
		return fmt.Errorf("session does not exist: %s", sessionName)
	}

	// Use -l flag to send keys literally (don't expand key bindings)
	cmd := exec.Command(a.tmuxPath, "send-keys", "-t", sessionName, "-l", keys)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send keys to tmux session: %w (stderr: %s)", err, stderr.String())
	}

	// Send Enter key separately
	cmd = exec.Command(a.tmuxPath, "send-keys", "-t", sessionName, "Enter")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send Enter key: %w", err)
	}

	return nil
}

// CapturePane captures the content of the current pane
func (a *RealAdapter) CapturePane(sessionName string) (string, error) {
	// Use CapturePaneWithOptions with 0 to capture all lines
	return a.CapturePaneWithOptions(sessionName, 0)
}

// AttachSession attaches to a tmux session
func (a *RealAdapter) AttachSession(sessionName string) error {
	// This needs to be run in the user's terminal, not from within the process
	// We'll return the command that should be executed
	return fmt.Errorf("to attach to session, run: tmux attach-session -t %s", sessionName)
}

// ListSessions returns a list of active tmux sessions
func (a *RealAdapter) ListSessions() ([]string, error) {
	cmd := exec.Command(a.tmuxPath, "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// No sessions returns exit code 1
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list tmux sessions: %w", err)
	}

	// Parse output
	sessions := []string{}
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			sessions = append(sessions, line)
		}
	}

	return sessions, nil
}

// GetSessionPID gets the PID of the main process in a tmux session
func (a *RealAdapter) GetSessionPID(sessionName string) (int, error) {
	// Get the PID of the first pane in the session
	cmd := exec.Command(a.tmuxPath, "list-panes", "-t", sessionName, "-F", "#{pane_pid}")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to get session PID: %w", err)
	}

	// Parse the first PID
	pidStr := strings.TrimSpace(string(output))
	lines := strings.Split(pidStr, "\n")
	if len(lines) > 0 && lines[0] != "" {
		var pid int
		if _, err := fmt.Sscanf(lines[0], "%d", &pid); err == nil {
			return pid, nil
		}
	}

	return 0, fmt.Errorf("no PID found for session")
}

// SetEnvironment sets environment variables in a tmux session
func (a *RealAdapter) SetEnvironment(sessionName string, env map[string]string) error {
	for key, value := range env {
		// Set environment variable in tmux session
		// This will be available for new panes/windows
		cmd := exec.Command(a.tmuxPath, "set-environment", "-t", sessionName, key, value)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set environment %s: %w", key, err)
		}
	}
	return nil
}

// ResizeWindow resizes the tmux window to standard dimensions
func (a *RealAdapter) ResizeWindow(sessionName string, width, height int) error {
	// Resize the window
	cmd := exec.Command(a.tmuxPath, "resize-window", "-t", sessionName, "-x", fmt.Sprintf("%d", width), "-y", fmt.Sprintf("%d", height))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Ignore resize errors as they're not critical
		// Some terminals might not support exact sizes
		return nil //nolint:nilerr // Intentionally ignoring resize errors
	}
	return nil
}

// CapturePaneWithOptions captures the content with specified options
func (a *RealAdapter) CapturePaneWithOptions(sessionName string, lines int) (string, error) {
	// Build command with options
	// -p: print to stdout
	// -J: join wrapped lines
	// -e: include escape sequences (ANSI color codes)
	args := []string{"capture-pane", "-t", sessionName, "-p", "-J", "-e"}

	// Add line limit if specified
	if lines > 0 {
		// Capture last N lines
		args = append(args, "-S", fmt.Sprintf("-%d", lines))
	}

	cmd := exec.Command(a.tmuxPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture pane: %w", err)
	}
	return string(output), nil
}

// IsPaneDead checks if the pane's process has exited
func (a *RealAdapter) IsPaneDead(sessionName string) (bool, error) {
	// Get pane_dead status - returns "1" if dead, "0" if alive
	cmd := exec.Command(a.tmuxPath, "list-panes", "-t", sessionName, "-F", "#{pane_dead}")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check pane status: %w", err)
	}

	// Parse output
	result := strings.TrimSpace(string(output))
	return result == "1", nil
}
