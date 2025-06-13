package session

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/terminal"
	"github.com/aki/amux/internal/core/workspace"
)

// tmuxSessionImpl implements Session interface with tmux backend
type tmuxSessionImpl struct {
	info        *Info
	store       Store
	tmuxAdapter tmux.Adapter
	workspace   *workspace.Workspace
	logger      logger.Logger
	mu          sync.RWMutex
}

// TmuxSessionOption is a function that configures a tmux session
type TmuxSessionOption func(*tmuxSessionImpl)

// WithTmuxLogger sets the logger for the tmux session
func WithTmuxLogger(log logger.Logger) TmuxSessionOption {
	return func(s *tmuxSessionImpl) {
		s.logger = log
	}
}

// NewTmuxSession creates a new tmux-backed session
func NewTmuxSession(info *Info, store Store, tmuxAdapter tmux.Adapter, workspace *workspace.Workspace, opts ...TmuxSessionOption) Session {
	s := &tmuxSessionImpl{
		info:        info,
		store:       store,
		tmuxAdapter: tmuxAdapter,
		workspace:   workspace,
		logger:      logger.Nop(), // Default to no-op logger
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
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
	return s.info.Status
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

	if s.info.Status == StatusRunning {
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
	s.info.Status = StatusRunning
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

	if s.info.Status != StatusRunning {
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
	s.info.Status = StatusStopped
	s.info.StoppedAt = &now

	if err := s.store.Save(s.info); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (s *tmuxSessionImpl) Attach() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.info.Status != StatusRunning {
		return ErrSessionNotRunning{ID: s.info.ID}
	}

	if s.info.TmuxSession == "" {
		return fmt.Errorf("no tmux session associated")
	}

	// Return instruction for attaching
	return s.tmuxAdapter.AttachSession(s.info.TmuxSession)
}

func (s *tmuxSessionImpl) SendInput(input string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.info.Status != StatusRunning {
		return ErrSessionNotRunning{ID: s.info.ID}
	}

	if s.info.TmuxSession == "" {
		return fmt.Errorf("no tmux session associated")
	}

	return s.tmuxAdapter.SendKeys(s.info.TmuxSession, input)
}

func (s *tmuxSessionImpl) GetOutput() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.info.Status != StatusRunning {
		return nil, ErrSessionNotRunning{ID: s.info.ID}
	}

	if s.info.TmuxSession == "" {
		return nil, fmt.Errorf("no tmux session associated")
	}

	output, err := s.tmuxAdapter.CapturePane(s.info.TmuxSession)
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
