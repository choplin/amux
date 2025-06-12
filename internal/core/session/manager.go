package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	contextmgr "github.com/aki/amux/internal/core/context"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/mailbox"
	"github.com/aki/amux/internal/core/workspace"
)

// Manager implements Manager interface
type Manager struct {
	store            Store
	workspaceManager *workspace.Manager
	mailboxManager   *mailbox.Manager
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
func NewManager(store Store, workspaceManager *workspace.Manager, mailboxManager *mailbox.Manager, idMapper *idmap.IDMapper, opts ...ManagerOption) *Manager {
	// Try to create tmux adapter, but don't fail if unavailable
	tmuxAdapter, _ := tmux.NewAdapter()

	m := &Manager{
		store:            store,
		workspaceManager: workspaceManager,
		mailboxManager:   mailboxManager,
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

	// Generate unique session ID
	sessionID := generateSessionID()

	// Set defaults
	if opts.Command == "" {
		opts.Command = "bash"
	}
	if opts.AgentID == "" {
		opts.AgentID = "default"
	}

	now := time.Now()
	info := &Info{
		ID:              sessionID,
		WorkspaceID:     ws.ID,
		AgentID:         opts.AgentID,
		Status:          StatusCreated,
		Command:         opts.Command,
		Environment:     opts.Environment,
		InitialPrompt:   opts.InitialPrompt,
		CreatedAt:       now,
		StatusChangedAt: now,
	}

	// Save session info to store
	if err := m.store.Save(info); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Register the session ID for short ID mapping
	if err := m.idMapper.Register(sessionID); err != nil {
		// Log warning but don't fail
		m.logger.WithField("session_id", sessionID).Warn("Failed to register session ID")
	}

	// Update index after registration
	if index, err := m.idMapper.Resolve(sessionID); err == nil {
		info.Index = index.Index
		// Update the stored info with index
		if err := m.store.Save(info); err != nil {
			m.logger.WithField("session_id", sessionID).Warn("Failed to update session with index")
		}
	}

	// Create context in working directory if template exists
	if opts.CreateContext {
		// Create context manager
		contextManager := contextmgr.NewManager(
			contextmgr.WithWorkspaceID(ws.ID),
			contextmgr.WithSessionID(sessionID),
			contextmgr.WithAgentID(opts.AgentID),
		)

		// Try to create context
		if _, err := contextManager.CreateContext(ws.Path); err != nil {
			// Log warning but don't fail session creation
			m.logger.WithField("error", err).Warn("Failed to create working context")
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
	// Get all session IDs from store
	ids, err := m.store.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessions := make([]Session, 0, len(ids))
	for _, id := range ids {
		sess, err := m.GetSession(id)
		if err != nil {
			// Log warning but continue
			m.logger.WithField("session_id", id).WithField("error", err).Warn("Failed to load session")
			continue
		}
		sessions = append(sessions, sess)
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

	// Unregister the ID mapping
	if err := m.idMapper.Unregister(fullID); err != nil {
		// Log warning but don't fail
		m.logger.WithField("session_id", fullID).Warn("Failed to unregister session ID")
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
	// Try to resolve as index first
	if resolvedID, err := m.idMapper.Resolve(idOrIndex); err == nil {
		return resolvedID.ID, nil
	}

	// Otherwise, assume it's already a full ID
	return idOrIndex, nil
}