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
	"github.com/aki/amux/internal/core/session/state"
	"github.com/aki/amux/internal/core/terminal"
	"github.com/aki/amux/internal/core/workspace"
)

// tmuxSessionImpl implements both Session and TerminalSession interfaces with tmux backend
type tmuxSessionImpl struct {
	info           *Info
	manager        *Manager
	tmuxAdapter    tmux.Adapter
	workspace      *workspace.Workspace
	agentConfig    *config.Agent
	logger         logger.Logger
	processChecker process.Checker
	stateManager   *state.Manager
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

// WithStateManager sets the state manager for the session
func WithStateManager(sm *state.Manager) TmuxSessionOption {
	return func(s *tmuxSessionImpl) {
		s.stateManager = sm
	}
}

// NewTmuxSession creates a new tmux-backed session
func NewTmuxSession(info *Info, manager *Manager, tmuxAdapter tmux.Adapter, workspace *workspace.Workspace, agentConfig *config.Agent, opts ...TmuxSessionOption) TerminalSession {
	s := &tmuxSessionImpl{
		info:           info,
		manager:        manager,
		tmuxAdapter:    tmuxAdapter,
		workspace:      workspace,
		agentConfig:    agentConfig,
		logger:         logger.Nop(),    // Default to no-op logger
		processChecker: process.Default, // Default to system process checker
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Activity tracking fields are now stored in state machine

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
	if s.workspace == nil {
		return ""
	}
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

	// Using TODO context for status check that doesn't need cancellation
	return s.getStatusLocked(context.TODO())
}

// getStatusLocked returns the current status without acquiring locks.
// This method should only be called when the lock is already held.
func (s *tmuxSessionImpl) getStatusLocked(ctx context.Context) Status {
	// Read status from state machine if available
	if s.stateManager != nil {
		stateData, err := s.stateManager.GetState(ctx)
		if err == nil && stateData != nil {
			return stateData.State
		}
	}

	// Fallback: default to created
	return state.StatusCreated
}

func (s *tmuxSessionImpl) Info() *Info {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	info := *s.info
	return &info
}

func (s *tmuxSessionImpl) StatusChangedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stateManager != nil {
		stateData, err := s.stateManager.GetState(context.TODO())
		if err == nil && stateData != nil {
			return stateData.UpdatedAt
		}
	}
	return time.Time{}
}

func (s *tmuxSessionImpl) LastActivityTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stateManager != nil {
		stateData, err := s.stateManager.GetState(context.TODO())
		if err == nil && stateData != nil {
			return stateData.LastOutputTime
		}
	}
	return time.Time{}
}

func (s *tmuxSessionImpl) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use state machine if available
	if s.stateManager != nil {
		// The state machine will handle validation and state transitions
		// Session may already be in Starting state (from CreateSession), check current state
		currentState, err := s.stateManager.GetState(ctx)
		if err != nil {
			return fmt.Errorf("failed to get current state: %w", err)
		}

		// If we're already in Starting state (from semaphore acquisition), continue
		// Otherwise, transition to Starting first
		if currentState.State != state.StatusStarting {
			if err := s.stateManager.TransitionTo(ctx, state.StatusStarting); err != nil {
				return err
			}
		}
	} else {
		// Legacy behavior for backward compatibility
		status := s.getStatusLocked(ctx)
		if status.IsRunning() {
			return ErrSessionAlreadyRunning{ID: s.info.ID}
		}

		// Cannot start orphaned session
		if status == state.StatusOrphaned {
			return fmt.Errorf("cannot start orphaned session: %s", s.info.Error)
		}
	}

	// Cannot start session without workspace
	if s.workspace == nil {
		return fmt.Errorf("cannot start session: workspace not available")
	}

	// Generate tmux session name
	tmuxSession := fmt.Sprintf("amux-%s-%s-%d",
		s.workspace.ID,
		s.info.AgentID,
		time.Now().Unix())

	// Get shell and window name from agent configuration
	var shell, windowName string
	if s.agentConfig != nil && s.agentConfig.Type == config.AgentTypeTmux {
		if tmuxParams, err := s.agentConfig.GetTmuxParams(); err == nil {
			shell = tmuxParams.Shell
			windowName = tmuxParams.WindowName
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

	// Transition to Running -> Working now that tmux session is created
	if s.stateManager != nil {
		// Starting -> Running -> Working
		if err := s.stateManager.TransitionTo(ctx, state.StatusRunning); err != nil {
			return fmt.Errorf("failed to transition to running: %w", err)
		}
		if err := s.stateManager.TransitionTo(ctx, state.StatusWorking); err != nil {
			return fmt.Errorf("failed to transition to working: %w", err)
		}
	}

	return nil
}

func (s *tmuxSessionImpl) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use state machine if available
	if s.stateManager != nil {
		// Transition to stopping state
		if err := s.stateManager.TransitionTo(ctx, state.StatusStopping); err != nil {
			return err
		}

		// Kill tmux session
		if s.info.TmuxSession != "" {
			if err := s.tmuxAdapter.KillSession(s.info.TmuxSession); err != nil {
				// Log error and transition to failed
				s.logger.Warn("failed to kill tmux session", "error", err, "session", s.info.TmuxSession)
				if err := s.stateManager.TransitionTo(ctx, state.StatusFailed); err != nil {
					s.logger.Debug("failed to transition to failed", "error", err)
				}
				return err
			}
		}

		// Transition to stopped
		if err := s.stateManager.TransitionTo(ctx, state.StatusStopped); err != nil {
			return err
		}

		// Update info to reflect state change
		now := time.Now()
		s.info.StoppedAt = &now

		// Save updated info
		return s.manager.Save(ctx, s.info)
	}

	// Legacy behavior
	status := s.getStatusLocked(ctx)
	if !status.IsRunning() {
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
	s.info.StoppedAt = &now

	if err := s.manager.Save(ctx, s.info); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	// Release workspace semaphore
	if err := s.manager.releaseSemaphore(ctx, s.info.ID, s.info.WorkspaceID); err != nil {
		// Log error but don't fail the stop operation
		s.logger.Warn("failed to release workspace semaphore", "error", err, "session_id", s.info.ID)
	}

	return nil
}

func (s *tmuxSessionImpl) Attach() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Use TODO context since we're in a read-locked method without context
	status := s.getStatusLocked(context.TODO())
	if !status.IsRunning() {
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

	// Use TODO context since we're in a read-locked method without context
	status := s.getStatusLocked(context.TODO())
	if !status.IsRunning() {
		return ErrSessionNotRunning{ID: s.info.ID}
	}

	if s.info.TmuxSession == "" {
		return fmt.Errorf("no tmux session associated")
	}

	// Send the input
	if err := s.tmuxAdapter.SendKeys(s.info.TmuxSession, input); err != nil {
		return err
	}

	// If status is idle, transition to working
	if s.stateManager != nil {
		stateData, err := s.stateManager.GetState(context.TODO())
		if err == nil && stateData != nil && stateData.State == state.StatusIdle {
			// Transition to working since we just sent input
			if err := s.stateManager.TransitionTo(context.Background(), state.StatusWorking); err != nil {
				s.logger.Debug("failed to transition to working after input", "error", err)
			}
		}
	}

	return nil
}

func (s *tmuxSessionImpl) GetOutput(maxLines int) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Use TODO context since we're in a read-locked method without context
	status := s.getStatusLocked(context.TODO())
	if !status.IsRunning() {
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
func (s *tmuxSessionImpl) UpdateStatus(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only update if session is running
	status := s.getStatusLocked(ctx)
	if !status.IsRunning() {
		return nil
	}

	// Check cache - skip update if checked recently
	if s.stateManager != nil {
		stateData, err := s.stateManager.GetState(ctx)
		if err == nil && stateData != nil && time.Since(stateData.LastStatusCheck) < statusCacheDuration {
			return nil
		}
	}

	// Last check time is updated in state machine when we update activity

	// Defer saving the session info at the end (single save point)
	defer func() {
		if err := s.manager.Save(ctx, s.info); err != nil {
			s.logger.Warn("failed to save session state", "error", err)
		}
	}()

	// Check if tmux session still exists
	if !s.tmuxAdapter.SessionExists(s.info.TmuxSession) {
		// Session doesn't exist anymore - mark as failed
		s.info.Error = "tmux session no longer exists"
		if s.stateManager != nil {
			_ = s.stateManager.TransitionTo(ctx, state.StatusFailed)
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
		s.info.Error = "shell process exited"
		if s.stateManager != nil {
			_ = s.stateManager.TransitionTo(ctx, state.StatusFailed)
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
			if exitCode != 0 {
				s.info.Error = fmt.Sprintf("command exited with code %d", exitCode)
				if s.stateManager != nil {
					if err := s.stateManager.TransitionTo(ctx, state.StatusFailed); err != nil {
						s.logger.Debug("failed to transition to failed", "error", err)
					}
				}
			} else {
				if s.stateManager != nil {
					if err := s.stateManager.TransitionTo(ctx, state.StatusCompleted); err != nil {
						s.logger.Debug("failed to transition to completed", "error", err)
					}
				}
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

	// Update output tracking and handle working/idle transitions
	if s.stateManager != nil {
		stateData, err := s.stateManager.GetState(ctx)
		if err == nil && stateData != nil {
			// Check if output changed
			if currentHash != stateData.LastOutputHash {
				// Output changed, agent is working
				if err := s.stateManager.UpdateActivity(ctx, currentHash, now); err != nil {
					s.logger.Warn("failed to update activity", "error", err)
				}

				// Transition to working if not already
				if stateData.State != state.StatusWorking {
					if err := s.stateManager.TransitionTo(ctx, state.StatusWorking); err != nil {
						s.logger.Debug("failed to transition to working", "error", err)
					}
				}
			} else {
				// No output change, check if we should transition to idle
				timeSinceLastOutput := time.Since(stateData.LastOutputTime)
				const idleThreshold = 3 * time.Second

				if timeSinceLastOutput >= idleThreshold && stateData.State == state.StatusWorking {
					// Transition to idle
					if err := s.stateManager.TransitionTo(ctx, state.StatusIdle); err != nil {
						s.logger.Debug("failed to transition to idle", "error", err)
					}
				}

				// Update last check time even if no output change
				if err := s.stateManager.UpdateActivity(ctx, stateData.LastOutputHash, stateData.LastOutputTime); err != nil {
					s.logger.Warn("failed to update activity check time", "error", err)
				}
			}
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
