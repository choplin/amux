package session

import (
	"context"
	"fmt"

	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/session/state"
)

// orphanedSessionImpl represents a session whose workspace no longer exists
type orphanedSessionImpl struct {
	info    *Info
	manager *Manager
	state.Manager
	logger logger.Logger
}

// CreateOrphanedSession creates a session in orphaned state
func CreateOrphanedSession(ctx context.Context, info *Info, manager *Manager, reason string) (Session, error) {
	s := &orphanedSessionImpl{
		info:    info,
		manager: manager,
		logger:  logger.Nop(),
	}

	// Initialize state manager
	if info.StateDir == "" {
		return nil, fmt.Errorf("CreateOrphanedSession: StateDir is required")
	}

	s.Manager = state.InitManager(info.ID, info.WorkspaceID, info.StateDir, nil)

	// Set to orphaned state
	if err := s.TransitionTo(ctx, state.StatusOrphaned); err != nil {
		// If we can't transition, it might already be orphaned
		s.logger.Warn("failed to transition to orphaned state", "error", err)
	}

	// Set error reason
	info.Error = reason

	return s, nil
}

func (s *orphanedSessionImpl) ID() string {
	return s.info.ID
}

func (s *orphanedSessionImpl) WorkspaceID() string {
	return s.info.WorkspaceID
}

func (s *orphanedSessionImpl) WorkspacePath() string {
	// Orphaned sessions have no workspace path
	return ""
}

func (s *orphanedSessionImpl) AgentID() string {
	return s.info.AgentID
}

func (s *orphanedSessionImpl) Type() Type {
	return s.info.Type
}

func (s *orphanedSessionImpl) Status() Status {
	// Always return orphaned
	return StatusOrphaned
}

func (s *orphanedSessionImpl) Info() *Info {
	// Return a copy to prevent external modification
	info := *s.info
	return &info
}

func (s *orphanedSessionImpl) Start(ctx context.Context) error {
	return fmt.Errorf("cannot start orphaned session: %s", s.info.Error)
}

func (s *orphanedSessionImpl) Stop(ctx context.Context) error {
	return fmt.Errorf("cannot stop orphaned session: %s", s.info.Error)
}

// OrphanedTerminalSession extends orphaned session with terminal capabilities
type OrphanedTerminalSession interface {
	Session
	TerminalSession
}

type orphanedTerminalSessionImpl struct {
	*orphanedSessionImpl
}

// CreateOrphanedTerminalSession creates a terminal session in orphaned state
func CreateOrphanedTerminalSession(ctx context.Context, info *Info, manager *Manager, reason string) (TerminalSession, error) {
	base, err := CreateOrphanedSession(ctx, info, manager, reason)
	if err != nil {
		return nil, err
	}

	return &orphanedTerminalSessionImpl{
		orphanedSessionImpl: base.(*orphanedSessionImpl),
	}, nil
}

func (s *orphanedTerminalSessionImpl) Attach() error {
	return fmt.Errorf("cannot attach to orphaned session: %s", s.info.Error)
}

func (s *orphanedTerminalSessionImpl) SendInput(input string) error {
	return fmt.Errorf("cannot send input to orphaned session: %s", s.info.Error)
}

func (s *orphanedTerminalSessionImpl) GetOutput(maxLines int) ([]byte, error) {
	return nil, fmt.Errorf("cannot get output from orphaned session: %s", s.info.Error)
}

func (s *orphanedTerminalSessionImpl) UpdateStatus(ctx context.Context) error {
	// Orphaned sessions don't need status updates
	return nil
}
