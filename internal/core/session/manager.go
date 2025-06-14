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
	ws, err := m.workspaceManager.ResolveWorkspace(opts.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Check if tmux is available
	if m.tmuxAdapter == nil || !m.tmuxAdapter.IsAvailable() {
		return nil, ErrTmuxNotAvailable{}
	}

	// Use provided ID or generate a new one
	var sessionID string
	if !opts.ID.IsEmpty() {
		sessionID = opts.ID.String()
	} else {
		sessionID = GenerateID().String()
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
	storagePath, err := m.store.CreateSessionStorage(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session storage: %w", err)
	}

	// Create session info
	info := &Info{
		ID:          sessionID,
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
	m.sessions[sessionID] = sess
	m.mu.Unlock()

	return sess, nil
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(idOrIndex string) (Session, error) {
	// Resolve to full ID
	fullID, err := m.resolveSessionID(idOrIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve session ID: %w", err)
	}

	// Check cache
	m.mu.RLock()
	if sess, ok := m.sessions[fullID]; ok {
		m.mu.RUnlock()
		return sess, nil
	}
	m.mu.RUnlock()

	// Load from store
	info, err := m.store.Load(fullID)
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
	m.sessions[fullID] = sess
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

// RemoveSession removes a session
func (m *Manager) RemoveSession(idOrIndex string) error {
	// Resolve to full ID
	fullID, err := m.resolveSessionID(idOrIndex)
	if err != nil {
		return fmt.Errorf("failed to resolve session ID: %w", err)
	}

	// Get session to check status
	sess, err := m.GetSession(fullID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is running
	if sess.Status().IsRunning() {
		return fmt.Errorf("cannot remove running session")
	}

	// Remove from store
	if err := m.store.Delete(fullID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Remove short ID mapping
	if m.idMapper != nil {
		_ = m.idMapper.RemoveSession(fullID)
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

// createSessionFromInfo creates the appropriate session implementation from stored info
func (m *Manager) createSessionFromInfo(info *Info) (Session, error) {
	// Check if tmux is available
	if m.tmuxAdapter == nil || !m.tmuxAdapter.IsAvailable() {
		return nil, ErrTmuxNotAvailable{}
	}

	// Get workspace for tmux session
	ws, err := m.workspaceManager.ResolveWorkspace(info.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("workspace not found for session: %w", err)
	}

	return NewTmuxSession(info, m.store, m.tmuxAdapter, ws), nil
}

// resolveSessionID resolves a short index or ID to full session ID
func (m *Manager) resolveSessionID(idOrIndex string) (string, error) {
	// Check if this is a short ID
	fullID := idOrIndex
	if m.idMapper != nil {
		if fullIDFromShort, exists := m.idMapper.GetSessionFull(idOrIndex); exists {
			fullID = fullIDFromShort
		}
	}
	return fullID, nil
}
