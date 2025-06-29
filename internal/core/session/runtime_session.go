package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session/state"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/runtime"
)

// runtimeSessionImpl implements Session interface using the Runtime API
type runtimeSessionImpl struct {
	info          *Info
	manager       *Manager
	state.Manager // Embedded state manager
	workspace     *workspace.Workspace
	agentConfig   *config.Agent
	runtime       runtime.Runtime
	process       runtime.Process
	mu            sync.RWMutex
}

// CreateRuntimeSession creates a new runtime-based session
func CreateRuntimeSession(ctx context.Context, info *Info, manager *Manager, rt runtime.Runtime, workspace *workspace.Workspace, agentConfig *config.Agent) (Session, error) {
	s := &runtimeSessionImpl{
		info:        info,
		manager:     manager,
		runtime:     rt,
		workspace:   workspace,
		agentConfig: agentConfig,
	}

	// Initialize state manager
	if info.StateDir == "" {
		return nil, fmt.Errorf("CreateRuntimeSession: StateDir is required")
	}

	// Initialize state manager
	s.Manager = state.InitManager(info.ID, info.WorkspaceID, info.StateDir, nil)

	// Add semaphore handler if workspace manager is available
	if manager != nil && manager.workspaceManager != nil {
		s.AddChangeHandler(createSemaphoreHandler(manager.workspaceManager))
	}

	return s, nil
}

// ID returns the unique session identifier
func (s *runtimeSessionImpl) ID() string {
	return s.info.ID
}

// WorkspaceID returns the workspace this session runs in
func (s *runtimeSessionImpl) WorkspaceID() string {
	return s.info.WorkspaceID
}

// WorkspacePath returns the workspace path
func (s *runtimeSessionImpl) WorkspacePath() string {
	if s.workspace == nil {
		return ""
	}
	return s.workspace.Path
}

// AgentID returns the agent running in this session
func (s *runtimeSessionImpl) AgentID() string {
	return s.info.AgentID
}

// Type returns the session type
func (s *runtimeSessionImpl) Type() Type {
	return s.info.Type
}

// Status returns the current session status
func (s *runtimeSessionImpl) Status() Status {
	// Get status from state manager
	currentState, err := s.CurrentState()
	if err != nil {
		return StatusFailed
	}
	return currentState
}

// Info returns the full session information
func (s *runtimeSessionImpl) Info() *Info {
	return s.info
}

// GetStoragePath returns the storage path for the session
func (s *runtimeSessionImpl) GetStoragePath() string {
	return s.info.StoragePath
}

// Start starts the session using the runtime
func (s *runtimeSessionImpl) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check current status
	currentStatus := s.Status()
	if currentStatus != StatusCreated {
		return fmt.Errorf("cannot start session in state: %s", currentStatus)
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
	if err := s.TransitionTo(ctx, state.StatusStarting); err != nil {
		return fmt.Errorf("failed to transition to starting state: %w", err)
	}

	// Convert agent config to ExecutionSpec
	spec := s.agentConfig.ToExecutionSpec()
	spec.WorkingDir = s.workspace.Path

	// Merge environment variables
	environment := make(map[string]string)
	// Add AMUX environment variables
	environment["AMUX_SESSION_ID"] = s.info.ID
	environment["AMUX_SESSION_INDEX"] = s.info.Index
	environment["AMUX_WORKSPACE_ID"] = s.workspace.ID
	environment["AMUX_WORKSPACE_PATH"] = s.workspace.Path
	environment["AMUX_AGENT_ID"] = s.info.AgentID
	// Merge agent environment
	for k, v := range spec.Environment {
		environment[k] = v
	}
	// Merge session environment (from CLI)
	for k, v := range s.info.Environment {
		environment[k] = v
	}
	spec.Environment = environment

	// Override command if specified
	if s.info.Command != "" {
		spec.Command = []string{"sh", "-c", s.info.Command}
	}

	// Execute using runtime
	process, err := s.runtime.Execute(ctx, spec)
	if err != nil {
		_ = s.TransitionTo(ctx, state.StatusFailed)
		return fmt.Errorf("failed to execute: %w", err)
	}

	s.process = process

	// Update info with process details
	s.info.PID = 0 // Runtime processes don't expose PID directly
	s.info.StartedAt = &[]time.Time{time.Now()}[0]

	// Save updated info
	if err := s.manager.Save(ctx, s.info); err != nil {
		// Log but don't fail
		_ = err
	}

	// Transition to running state
	if err := s.TransitionTo(ctx, state.StatusRunning); err != nil {
		// Try to stop the process
		_ = process.Stop(ctx)
		return fmt.Errorf("failed to transition to running state: %w", err)
	}

	// Monitor process completion in background
	go s.monitorProcess(context.Background())

	return nil
}

// Stop stops the session gracefully
func (s *runtimeSessionImpl) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check current status
	currentStatus := s.Status()
	if currentStatus != StatusRunning {
		return fmt.Errorf("cannot stop session in state: %s", currentStatus)
	}

	// Transition to stopping
	if err := s.TransitionTo(ctx, state.StatusStopping); err != nil {
		return fmt.Errorf("failed to transition to stopping state: %w", err)
	}

	// Stop the process
	if s.process != nil {
		if err := s.process.Stop(ctx); err != nil {
			// Force kill if stop fails
			_ = s.process.Kill(ctx)
		}
	}

	// Update info
	now := time.Now()
	s.info.StoppedAt = &now

	// Save updated info
	if err := s.manager.Save(ctx, s.info); err != nil {
		// Log but don't fail
		_ = err
	}

	// Transition to stopped
	if err := s.TransitionTo(ctx, state.StatusStopped); err != nil {
		return fmt.Errorf("failed to transition to stopped state: %w", err)
	}

	return nil
}

// monitorProcess monitors the process and updates status when it completes
func (s *runtimeSessionImpl) monitorProcess(ctx context.Context) {
	if s.process == nil {
		return
	}

	// Wait for process to complete
	err := s.process.Wait(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update status based on exit code
	exitCode, _ := s.process.ExitCode()
	if err != nil || exitCode != 0 {
		_ = s.TransitionTo(ctx, state.StatusFailed)
		s.info.Error = fmt.Sprintf("Process exited with code %d", exitCode)
	} else {
		_ = s.TransitionTo(ctx, state.StatusCompleted)
	}

	// Update info
	now := time.Now()
	s.info.StoppedAt = &now

	// Save updated info
	if err := s.manager.Save(ctx, s.info); err != nil {
		// Log but don't fail
		_ = err
	}
}
