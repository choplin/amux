package session

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/agent"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/filemanager"
)

// Manager implements Manager interface
type Manager struct {
	sessionsDir      string
	fileManager      *filemanager.Manager[Info]
	workspaceManager *workspace.Manager
	agentManager     *agent.Manager
	tmuxAdapter      tmux.Adapter
	sessions         map[string]Session
	idMapper         *idmap.IDMapper
	mu               sync.RWMutex
}

// ManagerOption is a function that configures a Manager
type ManagerOption func(*Manager)

// NewManager creates a new session manager
func NewManager(basePath string, workspaceManager *workspace.Manager, agentManager *agent.Manager, idMapper *idmap.IDMapper, opts ...ManagerOption) (*Manager, error) {
	// Ensure sessions directory exists
	sessionsDir := filepath.Join(basePath, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	// Try to create tmux adapter, but don't fail if unavailable
	tmuxAdapter, _ := tmux.NewAdapter()

	m := &Manager{
		sessionsDir:      sessionsDir,
		fileManager:      filemanager.NewManager[Info](),
		workspaceManager: workspaceManager,
		agentManager:     agentManager,
		idMapper:         idMapper,
		tmuxAdapter:      tmuxAdapter,
		sessions:         make(map[string]Session),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

// SetTmuxAdapter sets a custom tmux adapter (useful for testing)
func (m *Manager) SetTmuxAdapter(adapter tmux.Adapter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tmuxAdapter = adapter
}

// CreateSession creates a new session
func (m *Manager) CreateSession(ctx context.Context, opts Options) (Session, error) {
	// Validate workspace exists
	ws, err := m.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(opts.WorkspaceID))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Set default session type if not specified
	sessionType := opts.Type
	if sessionType == "" {
		sessionType = TypeTmux
	}

	// For now, we only support tmux sessions
	// In the future, this will be type-based
	if sessionType != TypeTmux {
		return nil, fmt.Errorf("unsupported session type: %s", sessionType)
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

	// Set default agent ID if not provided
	if opts.AgentID == "" {
		opts.AgentID = agent.DefaultAgentID
	}

	// Get agent configuration if available
	var agentConfig *config.Agent
	if m.agentManager != nil {
		if agent, err := m.agentManager.GetAgent(opts.AgentID); err == nil {
			agentConfig = agent

			// If no command specified, try to get it from agent config
			if opts.Command == "" && agent.Type == config.AgentTypeTmux {
				if params, err := agent.GetTmuxParams(); err == nil {
					if params.Command.IsArray() {
						// For array commands, join with spaces
						opts.Command = strings.Join(params.Command.Array, " ")
					} else if params.Command.Single != "" {
						opts.Command = params.Command.Single
					}
				}
			}
		}
	}

	now := time.Now()

	// Create session storage directory
	storagePath := filepath.Join(m.sessionsDir, sessionID.String(), "storage")
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create session storage: %w", err)
	}

	// Create state directory - consistent for all session types
	stateDir := filepath.Join(m.sessionsDir, sessionID.String(), "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Create session info
	info := &Info{
		ID:          sessionID.String(),
		Type:        sessionType,
		WorkspaceID: ws.ID,
		AgentID:     opts.AgentID,
		ActivityTracking: ActivityTracking{
			LastOutputTime: now,
		},
		Command:       opts.Command,
		Environment:   opts.Environment,
		InitialPrompt: opts.InitialPrompt,
		CreatedAt:     now,
		StoragePath:   storagePath,
		StateDir:      stateDir,
		Name:          opts.Name,
		Description:   opts.Description,
	}

	// Save session info to file
	if err := m.saveSessionInfo(ctx, info); err != nil {
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
	sess, err := CreateTmuxSession(ctx, info, m, m.tmuxAdapter, ws, agentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}
	m.mu.Lock()
	m.sessions[sessionID.String()] = sess
	m.mu.Unlock()

	return sess, nil
}

// Get retrieves a session by its full ID
func (m *Manager) Get(ctx context.Context, id ID) (Session, error) {
	// Check cache
	m.mu.RLock()
	if sess, ok := m.sessions[string(id)]; ok {
		m.mu.RUnlock()
		return sess, nil
	}
	m.mu.RUnlock()

	// Load from file
	info, err := m.loadSessionInfo(ctx, string(id))
	if err != nil {
		// If not found in file, it means the session doesn't exist
		// (even if it might have been in cache before)
		if os.IsNotExist(err) {
			return nil, ErrSessionNotFound{ID: string(id)}
		}
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	// Create session from info
	sess, err := m.createSessionFromInfo(ctx, info)
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
func (m *Manager) ListSessions(ctx context.Context) ([]Session, error) {
	// List all session infos
	infos, err := m.listSessionInfos(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessions := make([]Session, 0, len(infos))
	existingIDs := make([]string, 0, len(infos))
	for _, info := range infos {
		// Collect existing IDs for reconciliation
		existingIDs = append(existingIDs, info.ID)

		// Populate short ID
		if m.idMapper != nil {
			if index, exists := m.idMapper.GetSessionIndex(info.ID); exists {
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
		session, err := m.createSessionFromInfo(ctx, info)
		if err != nil {
			// Log warning but continue
			slog.Warn("Failed to create session", "session_id", info.ID, "error", err)
			continue
		}

		// Cache it
		m.mu.Lock()
		m.sessions[info.ID] = session
		m.mu.Unlock()

		sessions = append(sessions, session)
	}

	// Reconcile index state with actual sessions
	if m.idMapper != nil {
		if orphanedCount, err := m.idMapper.ReconcileSessions(existingIDs); err == nil && orphanedCount > 0 {
			slog.Debug("Cleaned up orphaned session indices", "count", orphanedCount)
		}
	}

	return sessions, nil
}

// Remove removes a session by its full ID
func (m *Manager) Remove(ctx context.Context, id ID) error {
	// Get session to check status
	sess, err := m.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is running
	if sess.Status().IsRunning() {
		return fmt.Errorf("cannot remove running session")
	}

	// Clean up any remaining tmux session
	info := sess.Info()
	if info.TmuxSession != "" && m.tmuxAdapter != nil && m.tmuxAdapter.IsAvailable() {
		// Check if tmux session exists before trying to kill it
		if m.tmuxAdapter.SessionExists(info.TmuxSession) {
			if err := m.tmuxAdapter.KillSession(info.TmuxSession); err != nil {
				// Log error but continue with removal
				slog.Warn("failed to kill tmux session during removal", "error", err, "session", info.TmuxSession)
			}
		}
	}

	// Remove session info file
	if err := m.deleteSessionInfo(ctx, string(id)); err != nil {
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
func (m *Manager) createSessionFromInfo(ctx context.Context, info *Info) (Session, error) {
	// Type is required
	if info.Type == "" {
		return nil, fmt.Errorf("session type is required")
	}

	// Create session based on type
	switch info.Type {
	case TypeTmux:
		// Check if tmux is available
		if m.tmuxAdapter == nil || !m.tmuxAdapter.IsAvailable() {
			return nil, ErrTmuxNotAvailable{}
		}

		// Try to get workspace for tmux session
		var ws *workspace.Workspace
		if m.workspaceManager != nil {
			var err error
			ws, err = m.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(info.WorkspaceID))
			if err != nil {
				// Workspace not found - create orphaned session
				reason := fmt.Sprintf("workspace not found: %s", info.WorkspaceID)
				info.Error = reason
				// Update the stored session info
				if updateErr := m.saveSessionInfo(ctx, info); updateErr != nil {
					slog.Warn("failed to update orphaned session info", "error", updateErr)
				}
				// Return orphaned terminal session
				sess, err := CreateOrphanedTerminalSession(ctx, info, m, reason)
				if err != nil {
					return nil, fmt.Errorf("failed to create orphaned session: %w", err)
				}
				return sess, nil
			}
		}

		// Get agent configuration if available
		var agentConfig *config.Agent
		if m.agentManager != nil {
			if agent, err := m.agentManager.GetAgent(info.AgentID); err == nil {
				agentConfig = agent
			}
		}

		sess, err := CreateTmuxSession(ctx, info, m, m.tmuxAdapter, ws, agentConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create tmux session: %w", err)
		}
		return sess, nil

	default:
		return nil, fmt.Errorf("unsupported session type: %s", info.Type)
	}
}

// ResolveSession resolves a session identifier (ID, index, or name) to a Session
func (m *Manager) ResolveSession(ctx context.Context, identifier Identifier) (Session, error) {
	// 1. Try as full ID
	session, err := m.Get(ctx, ID(identifier))
	if err == nil {
		return session, nil
	}

	// Store the original error for ID lookup
	var idErr error
	if _, ok := err.(ErrSessionNotFound); ok {
		idErr = err
	}

	// 2. Try as index (short ID)
	if m.idMapper != nil {
		if fullID, exists := m.idMapper.GetSessionFull(string(identifier)); exists {
			session, err := m.Get(ctx, ID(fullID))
			if err == nil {
				return session, nil
			}
		}
	}

	// 3. Try as name
	sessions, err := m.ListSessions(ctx)
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
		// If we had a clear "not found" error from ID lookup, return that
		if idErr != nil {
			return nil, idErr
		}
		return nil, fmt.Errorf("session not found: %s", identifier)
	case 1:
		return matches[0], nil
	default:
		// Multiple sessions with the same name
		return nil, fmt.Errorf("multiple sessions found with name '%s', please use ID instead", identifier)
	}
}

// UpdateAllStatuses updates the status of multiple sessions in parallel for better performance
func (m *Manager) UpdateAllStatuses(ctx context.Context, sessions []Session) {
	// Use goroutines to update statuses in parallel
	// This is beneficial because:
	// 1. Each session has its own mutex, so different sessions can update concurrently
	// 2. The main bottleneck is external process calls (tmux, pgrep), not the mutex
	// 3. I/O wait time can be utilized to process other sessions
	var wg sync.WaitGroup

	// Limit concurrency to avoid overwhelming the system with too many process calls
	semaphore := make(chan struct{}, 10) // Max 10 concurrent updates

	for _, sess := range sessions {
		if sess.Status().IsRunning() {
			wg.Add(1)
			go func(s Session) {
				defer wg.Done()
				semaphore <- struct{}{}        // Acquire
				defer func() { <-semaphore }() // Release

				// Try to update status if session supports terminal operations
				if terminalSess, ok := s.(TerminalSession); ok {
					_ = terminalSess.UpdateStatus(ctx) // Ignore errors, just use current status if update fails
				}
			}(sess)
		}
	}
	wg.Wait()
}

// Helper methods for file operations

// saveSessionInfo saves session info to file
func (m *Manager) saveSessionInfo(ctx context.Context, info *Info) error {
	path := m.getSessionPath(info.ID)
	return m.fileManager.Write(ctx, path, info)
}

// loadSessionInfo loads session info from file
func (m *Manager) loadSessionInfo(ctx context.Context, id string) (*Info, error) {
	path := m.getSessionPath(id)

	// Attempt migration first
	sessionFile := filepath.Join(path, "session.yaml")
	if err := MigrateSessionInfo(sessionFile); err != nil {
		// Log but don't fail - migration is best effort
		slog.Debug("failed to migrate session info", "session", id, "error", err)
	}

	info, _, err := m.fileManager.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	return info, nil
}

// listSessionInfos lists all session infos
func (m *Manager) listSessionInfos(ctx context.Context) ([]*Info, error) {
	entries, err := os.ReadDir(m.sessionsDir)
	if err != nil {
		return nil, err
	}

	var infos []*Info
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Try to load session info
		info, err := m.loadSessionInfo(ctx, entry.Name())
		if err != nil {
			// Skip invalid sessions
			continue
		}
		infos = append(infos, info)
	}

	return infos, nil
}

// deleteSessionInfo deletes session info and storage
func (m *Manager) deleteSessionInfo(ctx context.Context, id string) error {
	// Remove session info file
	path := m.getSessionPath(id)
	if err := m.fileManager.Delete(ctx, path); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove session directory with retry for Windows
	sessionDir := filepath.Join(m.sessionsDir, id)
	for i := 0; i < 3; i++ {
		err := os.RemoveAll(sessionDir)
		if err == nil || os.IsNotExist(err) {
			return nil
		}
		// On Windows, files may still be locked, wait a bit before retry
		if i < 2 {
			time.Sleep(50 * time.Millisecond)
		}
	}
	// Final attempt
	if err := os.RemoveAll(sessionDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove session directory: %w", err)
	}

	return nil
}

// updateSessionInfo safely updates session info using CAS
func (m *Manager) updateSessionInfo(ctx context.Context, id string, updateFunc func(info *Info) error) error {
	path := m.getSessionPath(id)
	return m.fileManager.Update(ctx, path, updateFunc)
}

// getSessionPath returns the path to a session's info file
func (m *Manager) getSessionPath(id string) string {
	return filepath.Join(m.sessionsDir, id, "session.yaml")
}

// Implement Store interface methods

// Save implements Store.Save
func (m *Manager) Save(ctx context.Context, info *Info) error {
	return m.saveSessionInfo(ctx, info)
}

// Load implements Store.Load
func (m *Manager) Load(ctx context.Context, id string) (*Info, error) {
	return m.loadSessionInfo(ctx, id)
}

// List implements Store.List
func (m *Manager) List(ctx context.Context) ([]*Info, error) {
	return m.listSessionInfos(ctx)
}

// Delete implements Store.Delete
func (m *Manager) Delete(ctx context.Context, id string) error {
	return m.deleteSessionInfo(ctx, id)
}

// Update implements Store.Update
func (m *Manager) Update(ctx context.Context, id string, updateFunc func(info *Info) error) error {
	return m.updateSessionInfo(ctx, id, updateFunc)
}

// CreateSessionStorage implements Store.CreateSessionStorage
func (m *Manager) CreateSessionStorage(sessionID string) (string, error) {
	storagePath := filepath.Join(m.sessionsDir, sessionID, "storage")
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		return "", err
	}
	return storagePath, nil
}
