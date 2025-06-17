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
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/process"
	"github.com/aki/amux/internal/core/terminal"
	"github.com/aki/amux/internal/core/workspace"
)

// tmuxSessionImpl implements both Session and TerminalSession interfaces with tmux backend
type tmuxSessionImpl struct {
	info           *Info
	manager        *Manager
	tmuxAdapter    tmux.Adapter
	workspace      *workspace.Workspace
	logger         logger.Logger
	processChecker process.Checker
	mu             sync.RWMutex
}

// statusCacheDuration defines how long to cache status before rechecking.
// This is set to 1 second to balance performance with idle detection accuracy.
// With a 3-second idle threshold, this ensures we can detect idle state within
// 3-4 seconds rather than up to 5 seconds (which would happen with 2-second cache).
const statusCacheDuration = 1 * time.Second

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
func NewTmuxSession(info *Info, manager *Manager, tmuxAdapter tmux.Adapter, workspace *workspace.Workspace, opts ...TmuxSessionOption) TerminalSession {
	s := &tmuxSessionImpl{
		info:           info,
		manager:        manager,
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

	// Ensure type is set
	if s.info.Type == "" {
		s.info.Type = TypeTmux
	}

	return s
}

func (s *tmuxSessionImpl) ID() string {
	return s.info.ID
}

func (s *tmuxSessionImpl) WorkspaceID() string {
	return s.info.WorkspaceID
}

func (s *tmuxSessionImpl) WorkspacePath() string {
	return s.workspace.Path
}

func (s *tmuxSessionImpl) AgentID() string {
	return s.info.AgentID
}

func (s *tmuxSessionImpl) Type() Type {
	return TypeTmux
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

	// Get shell and window name from agent configuration
	var shell, windowName string
	if s.manager.agentManager != nil {
		if agentConfig, err := s.manager.agentManager.GetAgent(s.info.AgentID); err == nil && agentConfig != nil {
			if agentConfig.Type == config.AgentTypeTmux {
				if tmuxParams, err := agentConfig.GetTmuxParams(); err == nil {
					shell = tmuxParams.Shell
					windowName = tmuxParams.WindowName
				}
			}
		}
	}

	// Merge environment variables:
	// 1. AMUX standard environment variables
	// 2. Session environment (from agent config and CLI -e)
	environment := mergeEnvironment(
		getAMUXEnvironment(s),
		s.info.Environment,
	)

	// Create tmux session with options
	opts := tmux.CreateSessionOptions{
		SessionName: tmuxSession,
		WorkDir:     s.workspace.Path,
		Shell:       shell,
		WindowName:  windowName,
		Environment: environment,
	}
	if err := s.tmuxAdapter.CreateSessionWithOptions(opts); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
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

	if err := s.manager.Save(ctx, s.info); err != nil {
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

	if err := s.manager.Save(context.TODO(), s.info); err != nil {
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
		if err := s.manager.Save(context.TODO(), s.info); err != nil {
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
	switch s.info.AgentID {
	case "claude":
		// Command to start Claude Code with the workspace
		return fmt.Sprintf("claude code --workspace %s", s.workspace.Path)
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

	// Check cache - skip update if checked recently
	if time.Since(s.info.StatusState.LastStatusCheck) < statusCacheDuration {
		return nil
	}

	// Update last check time
	s.info.StatusState.LastStatusCheck = time.Now()

	// Defer saving the session info at the end (single save point)
	defer func() {
		if err := s.manager.Save(context.TODO(), s.info); err != nil {
			s.logger.Warn("failed to save session state", "error", err)
		}
	}()

	// Check if tmux session still exists
	if !s.tmuxAdapter.SessionExists(s.info.TmuxSession) {
		// Session doesn't exist anymore - mark as failed
		now := time.Now()
		s.info.StatusState.Status = StatusFailed
		s.info.StatusState.StatusChangedAt = now
		s.info.Error = "tmux session no longer exists"
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
	} else {
		// No output change, check if we should transition to idle.
		// With 1-second cache duration and 3-second idle threshold,
		// idle detection happens within 3-4 seconds of last activity.
		timeSinceLastOutput := time.Since(s.info.StatusState.LastOutputTime)
		const idleThreshold = 3 * time.Second

		if timeSinceLastOutput >= idleThreshold && s.info.StatusState.Status == StatusWorking {
			// Transition to idle
			s.info.StatusState.Status = StatusIdle
			s.info.StatusState.StatusChangedAt = now
		}
	}

	return nil
}

// captureExitStatus captures the exit status from the shell and saves it to storage
func (s *tmuxSessionImpl) captureExitStatus() (int, error) {
	if s.info.StoragePath == "" {
		return 0, fmt.Errorf("no storage path configured")
	}

	exitStatusPath := filepath.Join(s.info.StoragePath, "exit_status")

	// Send command to write exit status directly to storage
	cmd := fmt.Sprintf("echo $? > %s", exitStatusPath)
	if err := s.tmuxAdapter.SendKeys(s.info.TmuxSession, cmd); err != nil {
		return 0, fmt.Errorf("failed to send exit status command: %w", err)
	}

	// Wait for the file to be written
	time.Sleep(100 * time.Millisecond)

	// Read the exit status from the file
	data, err := os.ReadFile(exitStatusPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read exit status: %w", err)
	}

	// Parse the exit code
	exitCodeStr := strings.TrimSpace(string(data))
	exitCode, err := strconv.Atoi(exitCodeStr)
	if err != nil {
		return 0, fmt.Errorf("invalid exit status format: %s", exitCodeStr)
	}

	return exitCode, nil
}
