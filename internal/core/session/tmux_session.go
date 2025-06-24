package session

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
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
	stateManager   *state.Manager
	tmuxAdapter    tmux.Adapter
	workspace      *workspace.Workspace
	agentConfig    *config.Agent
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

// CreateTmuxSession creates and initializes a new tmux-backed session
func CreateTmuxSession(ctx context.Context, info *Info, manager *Manager, tmuxAdapter tmux.Adapter, workspace *workspace.Workspace, agentConfig *config.Agent, opts ...TmuxSessionOption) TerminalSession {
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

	// Initialize state manager - StoragePath is required
	if info.StoragePath == "" {
		panic("CreateTmuxSession: StoragePath is required")
	}

	stateDir := filepath.Join(info.StoragePath, "state")
	// Convert logger to slog
	var slogger *slog.Logger
	if s.logger != nil {
		// Create a simple slog adapter
		slogger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	s.stateManager = state.NewManager(info.ID, info.WorkspaceID, stateDir, slogger)

	// Set initial state based on current status
	if s.info.StatusState.Status != "" {
		// Map old status to new state
		mappedStatus := mapOldStatusToNew(s.info.StatusState.Status)
		currentState, err := s.stateManager.CurrentState()

		// Only try to set state if it's different or there was an error reading current state
		if err != nil || currentState != mappedStatus {
			// For sessions being migrated, we need to handle the case where
			// they're in states that can't be reached through normal transitions
			if mappedStatus == state.StatusRunning && (err != nil || currentState == state.StatusCreated) {
				// Transition through intermediate states to reach Running
				_ = s.stateManager.TransitionTo(ctx, state.StatusStarting)
				_ = s.stateManager.TransitionTo(ctx, state.StatusRunning)
			} else {
				// Try direct transition, ignore error for invalid transitions
				_ = s.stateManager.TransitionTo(ctx, mappedStatus)
			}
		}
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

	// Get status from state manager if available
	if s.stateManager != nil {
		if currentState, err := s.stateManager.CurrentState(); err == nil {
			// Map state.Status back to session.Status
			return mapStateStatusToSession(currentState)
		}
	}

	// Fallback to StatusState for backward compatibility
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

	// Check current status - use internal state to avoid deadlock
	var currentStatus Status
	if s.stateManager != nil {
		if state, err := s.stateManager.CurrentState(); err == nil {
			currentStatus = mapStateStatusToSession(state)
		} else {
			currentStatus = s.info.StatusState.Status
		}
	} else {
		currentStatus = s.info.StatusState.Status
	}

	if currentStatus.IsRunning() {
		return ErrSessionAlreadyRunning{ID: s.info.ID}
	}

	// Cannot start orphaned session
	if currentStatus == StatusOrphaned {
		return fmt.Errorf("cannot start orphaned session: %s", s.info.Error)
	}

	// Cannot start session without workspace
	if s.workspace == nil {
		return fmt.Errorf("cannot start session: workspace not available")
	}

	// Transition to starting state
	if s.stateManager != nil {
		if err := s.stateManager.TransitionTo(ctx, state.StatusStarting); err != nil {
			return fmt.Errorf("failed to transition to starting state: %w", err)
		}
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

	// Transition to running state
	if s.stateManager != nil {
		if err := s.stateManager.TransitionTo(ctx, state.StatusRunning); err != nil {
			// Clean up on failure
			if killErr := s.tmuxAdapter.KillSession(tmuxSession); killErr != nil {
				s.logger.Warn("failed to kill tmux session during cleanup", "error", killErr, "session", tmuxSession)
			}
			return fmt.Errorf("failed to transition to running state: %w", err)
		}
	}

	// Update session info
	now := time.Now()
	s.info.StatusState.Status = StatusRunning // Update for backward compatibility
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

func (s *tmuxSessionImpl) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check current status - use internal state to avoid deadlock
	var currentStatus Status
	if s.stateManager != nil {
		if state, err := s.stateManager.CurrentState(); err == nil {
			currentStatus = mapStateStatusToSession(state)
		} else {
			currentStatus = s.info.StatusState.Status
		}
	} else {
		currentStatus = s.info.StatusState.Status
	}

	if !currentStatus.IsRunning() {
		return ErrSessionNotRunning{ID: s.info.ID}
	}

	// Transition to stopping state
	if s.stateManager != nil {
		if err := s.stateManager.TransitionTo(ctx, state.StatusStopping); err != nil {
			return fmt.Errorf("failed to transition to stopping state: %w", err)
		}
	}

	// Kill tmux session
	if s.info.TmuxSession != "" {
		if err := s.tmuxAdapter.KillSession(s.info.TmuxSession); err != nil {
			// Log error but continue
			s.logger.Warn("failed to kill tmux session", "error", err, "session", s.info.TmuxSession)
		}
	}

	// Transition to stopped state
	if s.stateManager != nil {
		if err := s.stateManager.TransitionTo(ctx, state.StatusStopped); err != nil {
			return fmt.Errorf("failed to transition to stopped state: %w", err)
		}
	}

	// Update status for backward compatibility
	now := time.Now()
	s.info.StatusState.Status = StatusStopped
	s.info.StatusState.StatusChangedAt = now
	s.info.StoppedAt = &now

	if err := s.manager.Save(ctx, s.info); err != nil {
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

	// Update last activity time for activity tracking
	now := time.Now()
	s.info.StatusState.LastOutputTime = now

	// Save the updated info
	if err := s.manager.Save(context.Background(), s.info); err != nil {
		// Log error but don't fail the input operation
		s.logger.Warn("failed to save activity update after input", "error", err)
	}

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
func (s *tmuxSessionImpl) UpdateStatus(ctx context.Context) error {
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
		if err := s.manager.Save(ctx, s.info); err != nil {
			s.logger.Warn("failed to save session state", "error", err)
		}
	}()

	// Check if tmux session still exists
	if !s.tmuxAdapter.SessionExists(s.info.TmuxSession) {
		// Session doesn't exist anymore - mark as failed
		if s.stateManager != nil {
			if err := s.stateManager.TransitionTo(ctx, state.StatusFailed); err != nil {
				s.logger.Warn("failed to transition to failed state", "error", err)
			}
		}
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
		if s.stateManager != nil {
			if err := s.stateManager.TransitionTo(ctx, state.StatusFailed); err != nil {
				s.logger.Warn("failed to transition to failed state", "error", err)
			}
		}
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
				if s.stateManager != nil {
					if err := s.stateManager.TransitionTo(ctx, state.StatusFailed); err != nil {
						s.logger.Warn("failed to transition to failed state", "error", err)
					}
				}
				s.info.StatusState.Status = StatusFailed
				s.info.Error = fmt.Sprintf("command exited with code %d", exitCode)
			} else {
				if s.stateManager != nil {
					if err := s.stateManager.TransitionTo(ctx, state.StatusCompleted); err != nil {
						s.logger.Warn("failed to transition to completed state", "error", err)
					}
				}
				s.info.StatusState.Status = StatusCompleted
			}
			s.info.StatusState.StatusChangedAt = now
			return nil
		}
	}

	// If we get here, session is running with active processes
	// Update activity tracking (but no longer change status)

	// Get last 20 lines of output for activity tracking
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

	// Check if output changed for activity tracking
	if currentHash != s.info.StatusState.LastOutputHash {
		// Output changed, update activity tracking
		s.info.StatusState.LastOutputHash = currentHash
		s.info.StatusState.LastOutputTime = now
	}
	// Note: We no longer transition between Working/Idle states

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

// mapOldStatusToNew maps old status values to new state machine statuses
func mapOldStatusToNew(oldStatus Status) state.Status {
	switch oldStatus {
	case StatusCreated:
		return state.StatusCreated
	case StatusWorking, StatusIdle:
		// Both working and idle map to running
		return state.StatusRunning
	case StatusStarting:
		return state.StatusStarting
	case StatusRunning:
		return state.StatusRunning
	case StatusStopping:
		return state.StatusStopping
	case StatusCompleted:
		return state.StatusCompleted
	case StatusStopped:
		return state.StatusStopped
	case StatusFailed:
		return state.StatusFailed
	case StatusOrphaned:
		return state.StatusOrphaned
	default:
		// Default to created for unknown statuses
		return state.StatusCreated
	}
}

// mapStateStatusToSession maps state.Status back to session.Status
func mapStateStatusToSession(stateStatus state.Status) Status {
	switch stateStatus {
	case state.StatusCreated:
		return StatusCreated
	case state.StatusStarting:
		return StatusStarting
	case state.StatusRunning:
		return StatusRunning
	case state.StatusStopping:
		return StatusStopping
	case state.StatusCompleted:
		return StatusCompleted
	case state.StatusStopped:
		return StatusStopped
	case state.StatusFailed:
		return StatusFailed
	case state.StatusOrphaned:
		return StatusOrphaned
	default:
		return StatusCreated
	}
}
