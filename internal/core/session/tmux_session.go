package session

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/process"
	"github.com/aki/amux/internal/core/terminal"
	"github.com/aki/amux/internal/core/workspace"
)

// tmuxSessionImpl implements Session interface with tmux backend
type tmuxSessionImpl struct {
	info           *Info
	store          Store
	tmuxAdapter    tmux.Adapter
	workspace      *workspace.Workspace
	logger         logger.Logger
	processChecker process.Checker
	mu             sync.RWMutex
}

// TmuxSessionOption is a function that configures a tmux session
type TmuxSessionOption func(*tmuxSessionImpl)

// WithTmuxLogger sets the logger for the tmux session
func WithTmuxLogger(log logger.Logger) TmuxSessionOption {
	return func(s *tmuxSessionImpl) {
		s.logger = log
	}
}

// WithProcessChecker sets the process checker for the tmux session
func WithProcessChecker(checker process.Checker) TmuxSessionOption {
	return func(s *tmuxSessionImpl) {
		s.processChecker = checker
	}
}

// NewTmuxSession creates a new tmux-backed session
func NewTmuxSession(info *Info, store Store, tmuxAdapter tmux.Adapter, workspace *workspace.Workspace, opts ...TmuxSessionOption) Session {
	s := &tmuxSessionImpl{
		info:           info,
		store:          store,
		tmuxAdapter:    tmuxAdapter,
		workspace:      workspace,
		logger:         logger.Nop(),    // Default to no-op logger
		processChecker: process.Default, // Default to system process checker
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Initialize StatusState if not set (e.g., when loading from store)
	if s.info.StatusState.StatusChangedAt.IsZero() {
		s.info.StatusState.StatusChangedAt = time.Now()
	}
	if s.info.StatusState.LastOutputTime.IsZero() {
		s.info.StatusState.LastOutputTime = time.Now()
	}

	return s
}

func (s *tmuxSessionImpl) ID() string {
	return s.info.ID
}

func (s *tmuxSessionImpl) WorkspaceID() string {
	return s.info.WorkspaceID
}

func (s *tmuxSessionImpl) AgentID() string {
	return s.info.AgentID
}

func (s *tmuxSessionImpl) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.info.StatusState.Status
}

func (s *tmuxSessionImpl) Info() *Info {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	info := *s.info
	return &info
}

func (s *tmuxSessionImpl) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.info.StatusState.Status.IsRunning() {
		return ErrSessionAlreadyRunning{ID: s.info.ID}
	}

	// Generate tmux session name
	tmuxSession := fmt.Sprintf("amux-%s-%s-%d",
		s.workspace.ID,
		s.info.AgentID,
		time.Now().Unix())

	// Create tmux session
	if err := s.tmuxAdapter.CreateSession(tmuxSession, s.workspace.Path); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Set environment variables
	env := make(map[string]string)
	for k, v := range s.info.Environment {
		env[k] = v
	}

	// Add workspace-specific environment
	env["AMUX_WORKSPACE_ID"] = s.workspace.ID
	env["AMUX_WORKSPACE_PATH"] = s.workspace.Path
	env["AMUX_SESSION_ID"] = s.info.ID
	env["AMUX_AGENT_ID"] = s.info.AgentID
	env["AMUX_CONTEXT_PATH"] = fmt.Sprintf("%s/.amux/context", s.workspace.Path)

	if err := s.tmuxAdapter.SetEnvironment(tmuxSession, env); err != nil {
		// Clean up on failure
		if killErr := s.tmuxAdapter.KillSession(tmuxSession); killErr != nil {
			s.logger.Warn("failed to kill tmux session during cleanup", "error", killErr, "session", tmuxSession)
		}
		return fmt.Errorf("failed to set environment: %w", err)
	}

	// Resize to terminal dimensions or use defaults
	width, height := terminal.GetSize()
	if err := s.tmuxAdapter.ResizeWindow(tmuxSession, width, height); err != nil {
		// Log warning but don't fail - resize is not critical
		s.logger.Warn("failed to resize tmux window", "error", err, "session", tmuxSession)
	}

	// Get the command to run
	command := s.info.Command
	if command == "" {
		// Use default command based on agent
		command = s.getDefaultCommand()
	}

	// Send the command to start the agent
	if command != "" {
		if err := s.tmuxAdapter.SendKeys(tmuxSession, command); err != nil {
			// Clean up on failure
			if killErr := s.tmuxAdapter.KillSession(tmuxSession); killErr != nil {
				fmt.Printf("Warning: failed to kill tmux session during cleanup: %v\n", killErr)
			}
			return fmt.Errorf("failed to start agent: %w", err)
		}
	}

	// Send initial prompt if provided
	if s.info.InitialPrompt != "" {
		// Small delay to let the agent process start
		// The input is buffered by tmux, so it won't be lost
		time.Sleep(100 * time.Millisecond)

		if err := s.tmuxAdapter.SendKeys(tmuxSession, s.info.InitialPrompt); err != nil {
			// Log warning but don't fail - initial prompt is not critical
			s.logger.Warn("failed to send initial prompt", "error", err, "session", tmuxSession)
		}
	}

	// Get PID
	pid, _ := s.tmuxAdapter.GetSessionPID(tmuxSession)

	// Update session info
	now := time.Now()
	s.info.StatusState.Status = StatusWorking // Initially working when started
	s.info.StatusState.StatusChangedAt = now
	s.info.StatusState.LastOutputTime = now // Reset output tracking
	s.info.StatusState.LastOutputHash = 0   // Reset hash
	s.info.StartedAt = &now
	s.info.TmuxSession = tmuxSession
	s.info.PID = pid

	if err := s.store.Save(s.info); err != nil {
		// Clean up on failure
		if killErr := s.tmuxAdapter.KillSession(tmuxSession); killErr != nil {
			s.logger.Warn("failed to kill tmux session during cleanup", "error", killErr, "session", tmuxSession)
		}
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (s *tmuxSessionImpl) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.info.StatusState.Status.IsRunning() {
		return ErrSessionNotRunning{ID: s.info.ID}
	}

	// Kill tmux session
	if s.info.TmuxSession != "" {
		if err := s.tmuxAdapter.KillSession(s.info.TmuxSession); err != nil {
			// Log error but continue
			s.logger.Warn("failed to kill tmux session", "error", err, "session", s.info.TmuxSession)
		}
	}

	// Update status
	now := time.Now()
	s.info.StatusState.Status = StatusStopped
	s.info.StatusState.StatusChangedAt = now
	s.info.StoppedAt = &now

	if err := s.store.Save(s.info); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (s *tmuxSessionImpl) Attach() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.info.StatusState.Status.IsRunning() {
		return ErrSessionNotRunning{ID: s.info.ID}
	}

	if s.info.TmuxSession == "" {
		return fmt.Errorf("no tmux session associated")
	}

	// Return instruction for attaching
	return s.tmuxAdapter.AttachSession(s.info.TmuxSession)
}

func (s *tmuxSessionImpl) SendInput(input string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.info.StatusState.Status.IsRunning() {
		return ErrSessionNotRunning{ID: s.info.ID}
	}

	if s.info.TmuxSession == "" {
		return fmt.Errorf("no tmux session associated")
	}

	// Send the input
	if err := s.tmuxAdapter.SendKeys(s.info.TmuxSession, input); err != nil {
		return err
	}

	// Update status to working since we just sent input
	now := time.Now()
	if s.info.StatusState.Status == StatusIdle {
		s.info.StatusState.Status = StatusWorking
		s.info.StatusState.StatusChangedAt = now
		// Save status change
		if err := s.store.Save(s.info); err != nil {
			// Log error but don't fail the input operation
			s.logger.Warn("failed to save status change after input", "error", err)
		}
	}
	// Always update last activity time
	s.info.StatusState.LastOutputTime = now

	return nil
}

func (s *tmuxSessionImpl) GetOutput(maxLines int) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.info.StatusState.Status.IsRunning() {
		return nil, ErrSessionNotRunning{ID: s.info.ID}
	}

	if s.info.TmuxSession == "" {
		return nil, fmt.Errorf("no tmux session associated")
	}

	// Capture only the requested number of lines
	output, err := s.tmuxAdapter.CapturePaneWithOptions(s.info.TmuxSession, maxLines)
	if err != nil {
		return nil, err
	}

	return []byte(output), nil
}

// getDefaultCommand returns the default command for an agent
func (s *tmuxSessionImpl) getDefaultCommand() string {
	// Create context directory path
	contextDir := filepath.Join(s.workspace.Path, ".amux", "context")

	switch s.info.AgentID {
	case "claude":
		// Command to start Claude Code with the workspace
		return fmt.Sprintf("claude code --workspace %s --context %s", s.workspace.Path, contextDir)
	default:
		// No default command for unknown agents
		return ""
	}
}

// UpdateStatus checks the current output and updates the session status accordingly.
// This should be called by commands that need fresh status information.
func (s *tmuxSessionImpl) UpdateStatus() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only update if session is running
	if !s.info.StatusState.Status.IsRunning() {
		return nil
	}

	// Check if tmux session still exists
	if !s.tmuxAdapter.SessionExists(s.info.TmuxSession) {
		// Session doesn't exist anymore - mark as failed
		now := time.Now()
		s.info.StatusState.Status = StatusFailed
		s.info.StatusState.StatusChangedAt = now
		s.info.Error = "tmux session no longer exists"
		if err := s.store.Save(s.info); err != nil {
			return fmt.Errorf("failed to save status change: %w", err)
		}
		return nil
	}

	// Check if shell process is dead
	isDead, err := s.tmuxAdapter.IsPaneDead(s.info.TmuxSession)
	if err != nil {
		// Log error but continue - don't fail the entire status update
		s.logger.Warn("failed to check pane status", "error", err)
	} else if isDead {
		// Shell process is dead - mark as failed
		now := time.Now()
		s.info.StatusState.Status = StatusFailed
		s.info.StatusState.StatusChangedAt = now
		s.info.Error = "shell process exited"
		if err := s.store.Save(s.info); err != nil {
			return fmt.Errorf("failed to save status change: %w", err)
		}
		return nil
	}

	// Check if shell has child processes
	if s.info.PID > 0 && s.processChecker != nil {
		hasChildren, err := s.processChecker.HasChildren(s.info.PID)
		if err != nil {
			// Log error but continue
			s.logger.Warn("failed to check child processes", "error", err, "pid", s.info.PID)
		} else if !hasChildren {
			// No child processes - capture exit status from shell
			exitCode, err := s.captureExitStatus()
			if err != nil {
				// Failed to capture exit status - log but continue
				s.logger.Warn("failed to capture exit status", "error", err)
			}

			// Determine final status based on exit code
			now := time.Now()
			if exitCode != 0 {
				s.info.StatusState.Status = StatusFailed
				s.info.Error = fmt.Sprintf("command exited with code %d", exitCode)
			} else {
				s.info.StatusState.Status = StatusCompleted
			}
			s.info.StatusState.StatusChangedAt = now

			if err := s.store.Save(s.info); err != nil {
				return fmt.Errorf("failed to save status change: %w", err)
			}
			return nil
		}
	}

	// If we get here, session is running with active processes
	// Continue with existing working/idle detection logic

	// Get last 20 lines of output for status checking
	// This is sufficient to detect activity without fetching entire buffer
	const statusCheckLines = 20
	output, err := s.tmuxAdapter.CapturePaneWithOptions(s.info.TmuxSession, statusCheckLines)
	if err != nil {
		return fmt.Errorf("failed to capture pane: %w", err)
	}

	// Calculate hash of output for efficient comparison
	h := fnv.New32a()
	h.Write([]byte(output))
	currentHash := h.Sum32()
	now := time.Now()

	// Check if output changed
	if currentHash != s.info.StatusState.LastOutputHash {
		// Output changed, agent is working
		s.info.StatusState.LastOutputHash = currentHash
		s.info.StatusState.LastOutputTime = now
		if s.info.StatusState.Status != StatusWorking {
			s.info.StatusState.Status = StatusWorking
			s.info.StatusState.StatusChangedAt = now
		}
		// Always save when hash changes to persist the new hash
		if err := s.store.Save(s.info); err != nil {
			return fmt.Errorf("failed to save status change: %w", err)
		}
	} else {
		// No output change, check if we should transition to idle
		timeSinceLastOutput := time.Since(s.info.StatusState.LastOutputTime)
		const idleThreshold = 3 * time.Second

		if timeSinceLastOutput >= idleThreshold && s.info.StatusState.Status == StatusWorking {
			// Transition to idle
			s.info.StatusState.Status = StatusIdle
			s.info.StatusState.StatusChangedAt = now
			// Save status change
			if err := s.store.Save(s.info); err != nil {
				return fmt.Errorf("failed to save status change: %w", err)
			}
		}
	}

	return nil
}

// captureExitStatus captures the exit status from the shell and saves it to storage
func (s *tmuxSessionImpl) captureExitStatus() (int, error) {
	// Send command to get exit status
	if err := s.tmuxAdapter.SendKeys(s.info.TmuxSession, "echo $?"); err != nil {
		return 0, fmt.Errorf("failed to send echo command: %w", err)
	}

	// Wait a bit for the command to execute
	time.Sleep(100 * time.Millisecond)

	// Capture the output (last few lines should contain the exit code)
	output, err := s.tmuxAdapter.CapturePaneWithOptions(s.info.TmuxSession, 5)
	if err != nil {
		return 0, fmt.Errorf("failed to capture pane: %w", err)
	}

	// Parse the output to find the exit code
	// Look for a line that contains just a number
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if exitCode, err := strconv.Atoi(line); err == nil {
			// Save to storage if available
			if s.info.StoragePath != "" {
				exitStatusPath := filepath.Join(s.info.StoragePath, "exit_status")
				if err := os.WriteFile(exitStatusPath, []byte(line), 0o644); err != nil {
					s.logger.Warn("failed to save exit status", "error", err)
				}
			}
			return exitCode, nil
		}
	}

	return 0, fmt.Errorf("could not parse exit status from output")
}
