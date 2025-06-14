package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/workspace"
)

// Manager implements Manager interface
type Manager struct {
	store            Store
	workspaceManager *workspace.Manager
	tmuxAdapter      tmux.Adapter
	sessions         map[string]Session
	idMapper         *idmap.IDMapper
	logger           logger.Logger
	mu               sync.RWMutex
}

// ManagerOption is a function that configures a Manager
type ManagerOption func(*Manager)

// WithLogger sets the logger for the Manager
func WithLogger(log logger.Logger) ManagerOption {
	return func(m *Manager) {
		m.logger = log
	}
}

// NewManager creates a new session manager
func NewManager(store Store, workspaceManager *workspace.Manager, idMapper *idmap.IDMapper, opts ...ManagerOption) *Manager {
	// Try to create tmux adapter, but don't fail if unavailable
	tmuxAdapter, _ := tmux.NewAdapter()

	m := &Manager{
		store:            store,
		workspaceManager: workspaceManager,
		idMapper:         idMapper,
		tmuxAdapter:      tmuxAdapter,
		sessions:         make(map[string]Session),
		logger:           logger.Nop(), // Default to no-op logger
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// SetTmuxAdapter sets a custom tmux adapter (useful for testing)
func (m *Manager) SetTmuxAdapter(adapter tmux.Adapter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tmuxAdapter = adapter
}

// CreateSession creates a new session
func (m *Manager) CreateSession(opts Options) (Session, error) {
	// Validate workspace exists
	ws, err := m.workspaceManager.ResolveWorkspace(workspace.Identifier(opts.WorkspaceID))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Check if tmux is available
	if m.tmuxAdapter == nil || !m.tmuxAdapter.IsAvailable() {
		return nil, ErrTmuxNotAvailable{}
	}

	// Use provided ID or generate a new one
	var sessionID ID
	if !opts.ID.IsEmpty() {
		sessionID = opts.ID
	} else {
		sessionID = GenerateID()
	}

	// Set defaults
	if opts.Command == "" {
		opts.Command = "bash"
	}
	if opts.AgentID == "" {
		opts.AgentID = "default"
	}

	now := time.Now()

	// Create session storage directory
	storagePath, err := m.store.CreateSessionStorage(sessionID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create session storage: %w", err)
	}

	// Create session info
	info := &Info{
		ID:          sessionID.String(),
		WorkspaceID: ws.ID,
		AgentID:     opts.AgentID,
		StatusState: StatusState{
			Status:          StatusCreated,
			StatusChangedAt: now,
			LastOutputTime:  now,
		},
		Command:       opts.Command,
		Environment:   opts.Environment,
		InitialPrompt: opts.InitialPrompt,
		CreatedAt:     now,
		StoragePath:   storagePath,
		Name:          opts.Name,
		Description:   opts.Description,
	}

	// Save session info to store
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

	// Create and cache session
	sess := NewTmuxSession(info, m.store, m.tmuxAdapter, ws)
	m.mu.Lock()
	m.sessions[sessionID.String()] = sess
	m.mu.Unlock()

	return sess, nil
}

// Get retrieves a session by its full ID
func (m *Manager) Get(id ID) (Session, error) {
	// Check cache
	m.mu.RLock()
	if sess, ok := m.sessions[string(id)]; ok {
		m.mu.RUnlock()
		return sess, nil
	}
	m.mu.RUnlock()

	// Load from store
	info, err := m.store.Load(string(id))
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	// Create session from info
	sess, err := m.createSessionFromInfo(info)
	if err != nil {
		return nil, err
	}

	// Cache and return
	m.mu.Lock()
	m.sessions[string(id)] = sess
	m.mu.Unlock()

	return sess, nil
}

// ListSessions returns all sessions
func (m *Manager) ListSessions() ([]Session, error) {
	// Get all session infos from store
	infos, err := m.store.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
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

		// Create session from info
		session, err := m.createSessionFromInfo(info)
		if err != nil {
			// Log warning but continue
			m.logger.Warn("Failed to create session", "session_id", info.ID, "error", err)
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

// Remove removes a session by its full ID
func (m *Manager) Remove(id ID) error {
	// Get session to check status
	sess, err := m.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is running
	if sess.Status().IsRunning() {
		return fmt.Errorf("cannot remove running session")
	}

	// Remove from store
	if err := m.store.Delete(string(id)); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Remove short ID mapping
	if m.idMapper != nil {
		_ = m.idMapper.RemoveSession(string(id))
	}

	// Remove from cache
	m.mu.Lock()
	delete(m.sessions, string(id))
	m.mu.Unlock()

	return nil
}

// CleanupOrphaned cleans up orphaned sessions
func (m *Manager) CleanupOrphaned() error {
	// TODO: Implement orphaned session cleanup
	// This will check for tmux sessions without corresponding metadata
	return nil
}

// createSessionFromInfo creates the appropriate session implementation from stored info
func (m *Manager) createSessionFromInfo(info *Info) (Session, error) {
	// Check if tmux is available
	if m.tmuxAdapter == nil || !m.tmuxAdapter.IsAvailable() {
		return nil, ErrTmuxNotAvailable{}
	}

	// Get workspace for tmux session
	ws, err := m.workspaceManager.ResolveWorkspace(workspace.Identifier(info.WorkspaceID))
	if err != nil {
		return nil, fmt.Errorf("workspace not found for session: %w", err)
	}

	return NewTmuxSession(info, m.store, m.tmuxAdapter, ws), nil
}

// ResolveSession resolves a session identifier (ID, index, or name) to a Session
func (m *Manager) ResolveSession(identifier Identifier) (Session, error) {
	// 1. Try as full ID
	session, err := m.Get(ID(identifier))
	if err == nil {
		return session, nil
	}

	// 2. Try as index (short ID)
	if m.idMapper != nil {
		if fullID, exists := m.idMapper.GetSessionFull(string(identifier)); exists {
			session, err := m.Get(ID(fullID))
			if err == nil {
				return session, nil
			}
		}
	}

	// 3. Try as name
	sessions, err := m.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	var matches []Session
	for _, s := range sessions {
		if s.Info().Name == string(identifier) {
			matches = append(matches, s)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("session not found: %s", identifier)
	case 1:
		return matches[0], nil
	default:
		// Multiple sessions with the same name
		return nil, fmt.Errorf("multiple sessions found with name '%s', please use ID instead", identifier)
	}
}
