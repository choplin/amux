package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/common"
	contextmgr "github.com/aki/amux/internal/core/context"
	"github.com/aki/amux/internal/core/workspace"
)

// sessionImpl implements the Session interface
type sessionImpl struct {
	info  *SessionInfo
	store SessionStore
	mu    sync.RWMutex
}

func (s *sessionImpl) ID() string {
	return s.info.ID
}

func (s *sessionImpl) WorkspaceID() string {
	return s.info.WorkspaceID
}

func (s *sessionImpl) AgentID() string {
	return s.info.AgentID
}

func (s *sessionImpl) Status() SessionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.info.Status
}

func (s *sessionImpl) Info() *SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	info := *s.info
	return &info
}

func (s *sessionImpl) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.info.Status == StatusRunning {
		return ErrSessionAlreadyRunning{ID: s.info.ID}
	}

	// TODO: Implement actual session start with tmux
	// For now, just update status
	now := time.Now()
	s.info.Status = StatusRunning
	s.info.StartedAt = &now

	if err := s.store.Save(s.info); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (s *sessionImpl) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.info.Status != StatusRunning {
		return ErrSessionNotRunning{ID: s.info.ID}
	}

	// TODO: Implement actual session stop with tmux
	// For now, just update status
	now := time.Now()
	s.info.Status = StatusStopped
	s.info.StoppedAt = &now

	if err := s.store.Save(s.info); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (s *sessionImpl) Attach() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.info.Status != StatusRunning {
		return ErrSessionNotRunning{ID: s.info.ID}
	}

	// TODO: Implement actual attach with tmux
	return fmt.Errorf("attach not yet implemented")
}

func (s *sessionImpl) SendInput(input string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.info.Status != StatusRunning {
		return ErrSessionNotRunning{ID: s.info.ID}
	}

	// TODO: Implement actual input sending with tmux
	return fmt.Errorf("send input not yet implemented")
}

func (s *sessionImpl) GetOutput() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.info.Status != StatusRunning {
		return nil, ErrSessionNotRunning{ID: s.info.ID}
	}

	// TODO: Implement actual output capture with tmux
	return nil, fmt.Errorf("get output not yet implemented")
}

// Manager implements SessionManager interface
type Manager struct {
	store            SessionStore
	workspaceManager *workspace.Manager
	tmuxAdapter      tmux.Adapter
	sessions         map[string]Session
	idMapper         *common.IDMapper
	mu               sync.RWMutex
}

// NewManager creates a new session manager
func NewManager(store SessionStore, workspaceManager *workspace.Manager, idMapper *common.IDMapper) *Manager {
	// Try to create tmux adapter, but don't fail if unavailable
	tmuxAdapter, _ := tmux.NewAdapter()

	return &Manager{
		store:            store,
		workspaceManager: workspaceManager,
		idMapper:         idMapper,
		tmuxAdapter:      tmuxAdapter,
		sessions:         make(map[string]Session),
	}
}

// SetTmuxAdapter sets a custom tmux adapter (useful for testing)
func (m *Manager) SetTmuxAdapter(adapter tmux.Adapter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tmuxAdapter = adapter
}

// CreateSession creates a new session
func (m *Manager) CreateSession(opts SessionOptions) (Session, error) {
	// Validate workspace exists
	ws, err := m.workspaceManager.ResolveWorkspace(opts.WorkspaceID)
	if err != nil {
		return nil, ErrInvalidWorkspace{WorkspaceID: opts.WorkspaceID}
	}

	// TODO: Validate agent exists when we have agent configuration

	// Generate session ID
	id := fmt.Sprintf("session-%s-%d", generateID(), time.Now().Unix())

	// Create session info
	info := &SessionInfo{
		ID:          id,
		WorkspaceID: ws.ID,
		AgentID:     opts.AgentID,
		Status:      StatusCreated,
		Command:     opts.Command,
		Environment: opts.Environment,
		CreatedAt:   time.Now(),
	}

	// Initialize working context for the workspace
	contextManager := contextmgr.NewManager(ws.Path)
	if err := contextManager.Initialize(); err != nil {
		// Log error but don't fail session creation
		// Context is helpful but not critical
		fmt.Printf("Warning: failed to initialize working context: %v\n", err)
	} else {
		// Add initial log entry
		contextManager.AppendToWorkingLog(fmt.Sprintf("Session started for agent '%s'", opts.AgentID))
	}

	// Save to store
	if err := m.store.Save(info); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Generate and assign index
	if m.idMapper != nil {
		index, err := m.idMapper.AddSession(info.ID)
		if err != nil {
			// Don't fail if index generation fails
			info.Index = ""
		} else {
			info.Index = index
		}
	}

	// Create session implementation
	var session Session
	if m.tmuxAdapter != nil && m.tmuxAdapter.IsAvailable() {
		// Use tmux-backed session
		session = NewTmuxSession(info, m.store, m.tmuxAdapter, ws)
	} else {
		// Fall back to basic session
		session = &sessionImpl{
			info:  info,
			store: m.store,
		}
	}

	// Cache in memory
	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()

	return session, nil
}

// GetSession retrieves a session by ID (supports both short and full IDs)
func (m *Manager) GetSession(id string) (Session, error) {
	// Check if this is a short ID
	fullID := id
	if m.idMapper != nil {
		if fullIDFromShort, exists := m.idMapper.GetSessionFull(id); exists {
			fullID = fullIDFromShort
		}
	}

	// Check memory cache first
	m.mu.RLock()
	if session, ok := m.sessions[fullID]; ok {
		m.mu.RUnlock()
		return session, nil
	}
	m.mu.RUnlock()

	// Load from store
	info, err := m.store.Load(fullID)
	if err != nil {
		return nil, err
	}

	// Populate short ID
	if m.idMapper != nil {
		if index, exists := m.idMapper.GetSessionIndex(info.ID); exists {
			info.Index = index
		}
	}

	// Create session implementation
	session, err := m.createSessionFromInfo(info)
	if err != nil {
		return nil, err
	}

	// Cache in memory
	m.mu.Lock()
	m.sessions[info.ID] = session
	m.mu.Unlock()

	return session, nil
}

// ListSessions lists all sessions
func (m *Manager) ListSessions() ([]Session, error) {
	infos, err := m.store.List()
	if err != nil {
		return nil, err
	}

	sessions := make([]Session, 0, len(infos))
	for _, info := range infos {
		// Populate short ID
		if m.idMapper != nil {
			if index, exists := m.idMapper.GetSessionIndex(info.ID); exists {
				info.Index = index
			} else {
				// Generate index if it doesn't exist
				index, _ := m.idMapper.AddSession(info.ID)
				info.Index = index
			}
		}

		// Check if already in cache
		m.mu.RLock()
		if session, ok := m.sessions[info.ID]; ok {
			sessions = append(sessions, session)
			m.mu.RUnlock()
			continue
		}
		m.mu.RUnlock()

		// Create new session implementation
		session, err := m.createSessionFromInfo(info)
		if err != nil {
			// Skip sessions that can't be created
			continue
		}

		// Cache it
		m.mu.Lock()
		m.sessions[info.ID] = session
		m.mu.Unlock()

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// RemoveSession removes a stopped session
func (m *Manager) RemoveSession(id string) error {
	session, err := m.GetSession(id)
	if err != nil {
		return err
	}

	// Check if session is stopped
	if session.Status() == StatusRunning {
		return fmt.Errorf("cannot remove running session")
	}

	fullID := session.ID()

	// Remove from store
	if err := m.store.Delete(fullID); err != nil {
		return err
	}

	// Remove short ID mapping
	if m.idMapper != nil {
		if err := m.idMapper.RemoveSession(fullID); err != nil {
			// Don't fail if mapping removal fails
		}
	}

	// Remove from cache
	m.mu.Lock()
	delete(m.sessions, fullID)
	m.mu.Unlock()

	return nil
}

// CleanupOrphaned cleans up orphaned sessions
func (m *Manager) CleanupOrphaned() error {
	// TODO: Implement orphaned session cleanup
	// This will check for tmux sessions without corresponding metadata
	return nil
}

// generateID generates a short random ID
func generateID() string {
	// Simple ID generation for now
	// TODO: Use a proper ID generation library
	return fmt.Sprintf("%x", time.Now().UnixNano()%1000000)
}

// createSessionFromInfo creates the appropriate session implementation from stored info
func (m *Manager) createSessionFromInfo(info *SessionInfo) (Session, error) {
	// If we have tmux and the session was using tmux, create tmux session
	if m.tmuxAdapter != nil && m.tmuxAdapter.IsAvailable() && info.TmuxSession != "" {
		// Get workspace for tmux session
		ws, err := m.workspaceManager.ResolveWorkspace(info.WorkspaceID)
		if err != nil {
			// Workspace might be gone, fall back to basic session
			return &sessionImpl{
				info:  info,
				store: m.store,
			}, nil
		}
		return NewTmuxSession(info, m.store, m.tmuxAdapter, ws), nil
	}

	// Fall back to basic session
	return &sessionImpl{
		info:  info,
		store: m.store,
	}, nil

}
