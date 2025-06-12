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
		return nil, ErrInvalidWorkspace{WorkspaceID: opts.WorkspaceID}
	}

	// TODO: Validate agent exists when we have agent configuration

	// Use provided ID or generate a new one
	var sessionID string
	if !opts.ID.IsEmpty() {
		sessionID = opts.ID.String()
	} else {
		sessionID = GenerateID().String()
	}

	// Create session info
	info := &Info{
		ID:            sessionID,
		WorkspaceID:   ws.ID,
		AgentID:       opts.AgentID,
		Status:        StatusCreated,
		Command:       opts.Command,
		Environment:   opts.Environment,
		InitialPrompt: opts.InitialPrompt,
		CreatedAt:     time.Now(),
	}

	// Initialize working context for the workspace
	contextManager := contextmgr.NewManager(ws.Path)
	if err := contextManager.Initialize(); err != nil {
		// Log error but don't fail session creation
		// Context is helpful but not critical
		m.logger.Warn("failed to initialize working context", "error", err, "workspace", ws.ID)
	} else {
		// Add initial log entry
		if err := contextManager.AppendToWorkingLog(fmt.Sprintf("Session started for agent '%s'", opts.AgentID)); err != nil {
			// Log error but don't fail session creation
			m.logger.Warn("failed to append to working log", "error", err, "workspace", ws.ID)
		}
	}

	// Save to store
	if err := m.store.Save(info); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Initialize mailbox for the session
	if m.mailboxManager != nil {
		if err := m.mailboxManager.Initialize(sessionID); err != nil {
			// Log error but don't fail session creation
			m.logger.Warn("failed to initialize mailbox", "error", err, "session", sessionID)
		}
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

	// Check if tmux is available
	if m.tmuxAdapter == nil || !m.tmuxAdapter.IsAvailable() {
		return nil, ErrTmuxNotAvailable{}
	}

	// Create tmux-backed session
	session := NewTmuxSession(info, m.store, m.tmuxAdapter, ws, WithTmuxLogger(m.logger))

	// Cache in memory
	m.mu.Lock()
	m.sessions[sessionID] = session
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
	if session.Status().IsRunning() {
		return fmt.Errorf("cannot remove running session")
	}

	fullID := session.ID()

	// Remove from store
	if err := m.store.Delete(fullID); err != nil {
		return err
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