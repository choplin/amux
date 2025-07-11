// Package session provides session management for amux
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/aki/amux/internal/config"
	"github.com/aki/amux/internal/idmap"
	"github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/runtime/proxy"
	"github.com/aki/amux/internal/task"
	"github.com/aki/amux/internal/workspace"
)

// WorkspaceManager defines the interface for workspace operations needed by session manager
type WorkspaceManager interface {
	Create(ctx context.Context, opts workspace.CreateOptions) (*workspace.Workspace, error)
}

// Status represents the current state of a session
type Status string

const (
	// StatusStarting indicates the session is being initialized
	StatusStarting Status = "starting"
	// StatusRunning indicates the session is actively running
	StatusRunning Status = "running"
	// StatusStopped indicates the session stopped normally
	StatusStopped Status = "stopped"
	// StatusFailed indicates the session failed or crashed
	StatusFailed Status = "failed"
	// StatusUnknown indicates the session state cannot be determined
	StatusUnknown Status = "unknown"
)

// Session represents an active runtime session
type Session struct {
	ID          string                 `json:"id" yaml:"id"`
	ShortID     string                 `json:"short_id,omitempty" yaml:"short_id,omitempty"`
	Name        string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	WorkspaceID string                 `json:"workspace_id" yaml:"workspace_id"`
	TaskName    string                 `json:"task_name" yaml:"task_name"`
	Runtime     string                 `json:"runtime" yaml:"runtime"`
	Status      Status                 `json:"status" yaml:"status"`
	StartedAt   time.Time              `json:"started_at" yaml:"started_at"`
	StoppedAt   *time.Time             `json:"stopped_at,omitempty" yaml:"stopped_at,omitempty"`
	ExitCode    *int                   `json:"exit_code,omitempty" yaml:"exit_code,omitempty"`
	Command     []string               `json:"command" yaml:"command"`
	Environment map[string]string      `json:"environment,omitempty" yaml:"environment,omitempty"`
	WorkingDir  string                 `json:"working_dir" yaml:"working_dir"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Activity tracking fields
	LastActivityAt time.Time `json:"last_activity_at" yaml:"last_activity_at"`

	// Logging configuration
	EnableLog bool `json:"enable_log" yaml:"enable_log"`

	// Socket path for output streaming
	SocketPath string `json:"socket_path,omitempty" yaml:"socket_path,omitempty"`
}

// Manager manages sessions across workspaces
type Manager interface {
	// Create starts a new session
	Create(ctx context.Context, opts CreateOptions) (*Session, error)

	// Get retrieves a session by ID
	Get(ctx context.Context, id string) (*Session, error)

	// List returns all sessions, optionally filtered by workspace
	List(ctx context.Context, workspaceID string) ([]*Session, error)

	// Stop gracefully stops a session
	Stop(ctx context.Context, id string) error

	// Kill forcefully terminates a session
	Kill(ctx context.Context, id string) error

	// Attach attaches to a running session
	Attach(ctx context.Context, id string) error

	// Logs returns the logs for a session
	Logs(ctx context.Context, id string, follow bool) (LogReader, error)

	// Remove deletes a stopped session
	Remove(ctx context.Context, id string) error

	// UpdateStatus updates the status of a session
	UpdateStatus(ctx context.Context, id string, status Status) error

	// SendInput sends input to a running session
	SendInput(ctx context.Context, id string, input string) error
}

// CreateOptions defines options for creating a session
type CreateOptions struct {
	WorkspaceID         string                 // Workspace to run in
	AutoCreateWorkspace bool                   // Auto-create workspace if not specified
	Name                string                 // Human-readable name for the session
	Description         string                 // Description of session purpose
	TaskName            string                 // Task to execute (optional)
	Command             []string               // Direct command (if no task)
	Runtime             string                 // Runtime to use (default: local)
	Environment         map[string]string      // Additional environment variables
	WorkingDir          string                 // Working directory override
	Metadata            map[string]interface{} // Additional metadata
	RuntimeOptions      runtime.RuntimeOptions // Runtime-specific options
	EnableLog           bool                   // Enable logging to file (default: false)
}

// LogReader provides access to session logs
type LogReader interface {
	// Read reads log data
	Read(p []byte) (n int, err error)
	// Close closes the log reader
	Close() error
}

// manager implements the Manager interface
type manager struct {
	mu               sync.RWMutex
	sessions         map[string]*Session
	store            Store
	runtimes         map[string]runtime.Runtime
	tasks            *task.Manager
	workspaceManager WorkspaceManager
	configManager    *config.Manager
	idMapper         *idmap.Mapper[idmap.SessionID]
}

// NewManager creates a new session manager
func NewManager(store Store, runtimes map[string]runtime.Runtime, tasks *task.Manager, workspaceManager WorkspaceManager, configManager *config.Manager) Manager {
	// Initialize session ID mapper if config manager is available
	var idMapper *idmap.Mapper[idmap.SessionID]
	if configManager != nil {
		mapper, err := idmap.NewSessionIDMapper(configManager.GetAmuxDir())
		if err != nil {
			// Log error but continue - sessions will use fallback ID generation
			// This is to maintain backward compatibility
			mapper = nil
		}
		idMapper = mapper
	}

	return &manager{
		sessions:         make(map[string]*Session),
		store:            store,
		runtimes:         runtimes,
		tasks:            tasks,
		workspaceManager: workspaceManager,
		configManager:    configManager,
		idMapper:         idMapper,
	}
}

// Create starts a new session
func (m *manager) Create(ctx context.Context, opts CreateOptions) (*Session, error) {
	// Generate session ID first to use in workspace name
	var sessionID string
	var shortID string

	if m.idMapper != nil {
		// Use ID mapper to get persistent short ID
		fullID := fmt.Sprintf("session-%s-%d-%s",
			opts.Name,
			time.Now().Unix(),
			generateRandomSuffix())

		index, err := m.idMapper.Add(idmap.SessionID(fullID))
		if err != nil {
			return nil, fmt.Errorf("failed to acquire session ID: %w", err)
		}

		sessionID = fullID
		shortID = index
	} else {
		// Fallback to simple counter-based ID
		m.mu.Lock()
		counter := len(m.sessions) + 1
		m.mu.Unlock()
		sessionID = fmt.Sprintf("session-%d", counter)
		shortID = fmt.Sprintf("%d", counter)
	}

	// Handle auto workspace creation
	if opts.AutoCreateWorkspace && opts.WorkspaceID == "" {
		if m.workspaceManager == nil {
			return nil, fmt.Errorf("workspace manager not available for auto-creation")
		}
		// Create workspace with name based on session ID
		// Extract numeric part from sessionID (e.g., "session-1" -> "1")
		workspaceName := sessionID // Use full session ID as workspace name
		workspaceDesc := fmt.Sprintf("Auto-created for %s", sessionID)

		ws, err := m.workspaceManager.Create(ctx, workspace.CreateOptions{
			Name:        workspaceName,
			Description: workspaceDesc,
			AutoCreated: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create auto-workspace: %w", err)
		}

		opts.WorkspaceID = ws.ID
	}

	// Validate runtime
	if opts.Runtime == "" {
		opts.Runtime = "local"
	}
	rt, ok := m.runtimes[opts.Runtime]
	if !ok {
		return nil, fmt.Errorf("runtime not found: %s", opts.Runtime)
	}

	// Build execution spec
	spec := runtime.ExecutionSpec{
		SessionID:   sessionID,
		WorkingDir:  opts.WorkingDir,
		Environment: opts.Environment,
		Options:     opts.RuntimeOptions,
		EnableLog:   opts.EnableLog,
	}

	// If task is specified, load it
	if opts.TaskName != "" {
		t, err := m.tasks.GetTask(opts.TaskName)
		if err != nil {
			return nil, fmt.Errorf("failed to get task: %w", err)
		}

		// Parse command template
		cmd, err := task.ParseCommand(t.Command, opts.Environment)
		if err != nil {
			return nil, fmt.Errorf("failed to parse command: %w", err)
		}
		spec.Command = cmd

		// Merge task environment
		if spec.Environment == nil {
			spec.Environment = make(map[string]string)
		}
		for k, v := range t.Env {
			if _, exists := spec.Environment[k]; !exists {
				spec.Environment[k] = v
			}
		}

		// Use task working dir if not overridden
		if spec.WorkingDir == "" && t.WorkingDir != "" {
			spec.WorkingDir = t.WorkingDir
		}
	} else if len(opts.Command) > 0 {
		spec.Command = opts.Command
	} else {
		return nil, fmt.Errorf("either task name or command must be specified")
	}

	// Use provided metadata
	metadata := opts.Metadata

	// Generate socket path for this session
	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		tmpDir = "/tmp"
	}
	socketPath := filepath.Join(tmpDir, fmt.Sprintf("amux-%s.sock", sessionID))

	session := &Session{
		ID:             sessionID,
		ShortID:        shortID,
		Name:           opts.Name,
		Description:    opts.Description,
		WorkspaceID:    opts.WorkspaceID,
		TaskName:       opts.TaskName,
		Runtime:        opts.Runtime,
		Status:         StatusRunning,
		StartedAt:      time.Now(),
		Command:        spec.Command,
		Environment:    spec.Environment,
		WorkingDir:     spec.WorkingDir,
		Metadata:       metadata,
		LastActivityAt: time.Now(),
		EnableLog:      opts.EnableLog,
		SocketPath:     socketPath,
	}

	// Store session BEFORE starting the process
	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()

	// Save to persistent store
	if err := m.store.Save(ctx, session); err != nil {
		// Try to clean up
		m.mu.Lock()
		delete(m.sessions, session.ID)
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Start the process AFTER saving the session
	_, err := rt.Execute(ctx, spec)
	if err != nil {
		// Clean up the session
		m.mu.Lock()
		delete(m.sessions, session.ID)
		m.mu.Unlock()
		_ = m.store.Remove(ctx, session.ID)
		return nil, fmt.Errorf("failed to execute: %w", err)
	}

	return session, nil
}

// Get retrieves a session by ID (either short ID or full ID)
func (m *manager) Get(ctx context.Context, id string) (*Session, error) {
	// First try direct lookup
	m.mu.RLock()
	session, ok := m.sessions[id]
	m.mu.RUnlock()

	if ok {
		return session, nil
	}

	// If using ID mapper, try to resolve short ID to full ID
	fullID := id
	if m.idMapper != nil {
		if resolvedID, found := m.idMapper.GetFull(id); found {
			fullID = string(resolvedID)

			// Try memory lookup again with full ID
			m.mu.RLock()
			session, ok = m.sessions[fullID]
			m.mu.RUnlock()

			if ok {
				return session, nil
			}
		}
	}

	// Try to load from store
	session, err := m.store.Load(ctx, fullID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	// Note: We can't reconnect to process after restart since we don't store process IDs anymore
	// This is a simplification - sessions will appear as stopped after restart

	return session, nil
}

// List returns all sessions
func (m *manager) List(ctx context.Context, workspaceID string) ([]*Session, error) {
	// Get all sessions from store
	sessions, err := m.store.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Update with in-memory sessions
	sessionMap := make(map[string]*Session)
	for _, s := range sessions {
		sessionMap[s.ID] = s
	}

	// Override with in-memory sessions (more up-to-date)
	m.mu.RLock()
	for _, s := range m.sessions {
		if workspaceID == "" || s.WorkspaceID == workspaceID {
			sessionMap[s.ID] = s
		}
	}
	m.mu.RUnlock()

	// Convert back to slice
	result := make([]*Session, 0, len(sessionMap))
	for _, s := range sessionMap {
		// Update session information from runtime
		if s.Status == StatusRunning || s.Status == StatusStarting {
			m.updateSessionFromRuntime(ctx, s)
		}

		// Set short ID if using ID mapper
		if m.idMapper != nil && s.ShortID == "" {
			if idx, found := m.idMapper.GetIndex(idmap.SessionID(s.ID)); found {
				s.ShortID = idx
			}
		}

		result = append(result, s)
	}

	return result, nil
}

// Stop gracefully stops a session
func (m *manager) Stop(ctx context.Context, id string) error {
	session, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	// Get runtime
	rt, ok := m.runtimes[session.Runtime]
	if !ok {
		return fmt.Errorf("runtime not found: %s", session.Runtime)
	}

	// Check if runtime supports stop
	if stopper, ok := rt.(runtime.StoppableRuntime); ok {
		if err := stopper.Stop(ctx, session.ID); err != nil {
			return fmt.Errorf("failed to stop session: %w", err)
		}

		// Update session status
		if err := m.UpdateStatus(ctx, session.ID, StatusStopped); err != nil {
			return fmt.Errorf("failed to update session status: %w", err)
		}

		return nil
	}

	return fmt.Errorf("stop not supported for runtime: %s", session.Runtime)
}

// Kill forcefully terminates a session
func (m *manager) Kill(ctx context.Context, id string) error {
	session, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	// Get runtime
	rt, ok := m.runtimes[session.Runtime]
	if !ok {
		return fmt.Errorf("runtime not found: %s", session.Runtime)
	}

	// Check if runtime supports kill
	if killer, ok := rt.(runtime.KillableRuntime); ok {
		if err := killer.Kill(ctx, session.ID); err != nil {
			return fmt.Errorf("failed to kill session: %w", err)
		}

		// Update session status
		if err := m.UpdateStatus(ctx, session.ID, StatusFailed); err != nil {
			return fmt.Errorf("failed to update session status: %w", err)
		}

		return nil
	}

	return fmt.Errorf("kill not supported for runtime: %s", session.Runtime)
}

// Attach attaches to a running session
func (m *manager) Attach(ctx context.Context, id string) error {
	session, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	if session.Status != StatusRunning {
		return fmt.Errorf("session is not running")
	}

	// Get runtime
	rt, ok := m.runtimes[session.Runtime]
	if !ok {
		return fmt.Errorf("runtime not found: %s", session.Runtime)
	}

	// Check if runtime supports attach
	if attacher, ok := rt.(runtime.AttachableRuntime); ok {
		if err := attacher.Attach(ctx, session.ID); err != nil {
			return fmt.Errorf("failed to attach to session: %w", err)
		}
		return nil
	}

	return fmt.Errorf("attach not supported for runtime: %s", session.Runtime)
}

// Logs returns the logs for a session
func (m *manager) Logs(ctx context.Context, id string, follow bool) (LogReader, error) {
	if _, err := m.Get(ctx, id); err != nil {
		return nil, err
	}

	if follow {
		// Follow mode not supported in current implementation
		// Could be implemented with socket connection in the future
		return nil, fmt.Errorf("follow mode not yet implemented")
	}

	// Always use file store for logs
	return m.store.GetLogs(ctx, id)
}

// Remove deletes a stopped session
func (m *manager) Remove(ctx context.Context, id string) error {
	session, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	if session.Status == StatusRunning {
		return fmt.Errorf("cannot remove running session")
	}

	// Release ID back to pool if using ID mapper
	if m.idMapper != nil {
		_ = m.idMapper.Remove(idmap.SessionID(session.ID))
		// ID cleanup is not critical for session removal
	}

	// Remove from memory
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()

	// Remove from store
	if err := m.store.Remove(ctx, id); err != nil {
		return fmt.Errorf("failed to remove session: %w", err)
	}

	return nil
}

// UpdateStatus updates the status of a session
func (m *manager) UpdateStatus(ctx context.Context, id string, status Status) error {
	session, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	m.mu.Lock()
	session.Status = status
	m.mu.Unlock()

	// Save to store
	if err := m.store.Save(ctx, session); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

// SendInput sends input to a running session
func (m *manager) SendInput(ctx context.Context, id string, input string) error {
	session, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	if session.Status != StatusRunning {
		return fmt.Errorf("session is not running (status: %s)", session.Status)
	}

	// Get runtime
	rt, ok := m.runtimes[session.Runtime]
	if !ok {
		return fmt.Errorf("runtime not found: %s", session.Runtime)
	}

	// Check if runtime supports input sending
	if sender, ok := rt.(runtime.InputSendingRuntime); ok {
		return sender.SendInput(ctx, session.ID, input)
	}

	return fmt.Errorf("runtime %s does not support input sending", session.Runtime)
}

// simpleLogReader is a basic implementation of LogReader
type simpleLogReader struct {
	reader interface{ Read([]byte) (int, error) }
}

func (r *simpleLogReader) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r *simpleLogReader) Close() error {
	if closer, ok := r.reader.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// updateSessionFromRuntime updates session information from runtime
func (m *manager) updateSessionFromRuntime(ctx context.Context, session *Session) {
	// For local and local-detached runtimes, read status from the proxy status file
	if session.Runtime == "local" || session.Runtime == "local-detached" {
		// Use config manager to get the correct amux directory
		if m.configManager == nil {
			// In tests, configManager might be nil
			return
		}
		amuxDir := m.configManager.GetAmuxDir()
		statusPath := filepath.Join(amuxDir, "sessions", session.ID, "status.yaml")
		// Read status file if it exists
		if data, err := os.ReadFile(statusPath); err == nil {
			var status proxy.Status
			if err := yaml.Unmarshal(data, &status); err == nil {
				// Update session based on proxy status
				switch status.Status {
				case "running":
					session.Status = StatusRunning
					session.LastActivityAt = status.LastActivityAt
				case "exited":
					session.Status = StatusStopped
					if session.StoppedAt == nil {
						session.StoppedAt = &status.EndedAt
					}
					session.ExitCode = &status.ExitCode
				default:
					// Unknown status, keep current
				}

				// Update in memory and save if status changed
				m.mu.Lock()
				m.sessions[session.ID] = session
				m.mu.Unlock()
				_ = m.store.Save(ctx, session)
				return
			}
		}

		// If no status file or can't read it, mark as stopped
		session.Status = StatusStopped
		if session.StoppedAt == nil {
			now := time.Now()
			session.StoppedAt = &now
		}
		// Update in memory
		m.mu.Lock()
		m.sessions[session.ID] = session
		m.mu.Unlock()
		// Save to disk
		_ = m.store.Save(ctx, session)
		return
	}

	// For other runtimes (e.g., tmux), use runtime-specific logic
	rt, ok := m.runtimes[session.Runtime]
	if !ok {
		return
	}

	// Try to find the process by session ID
	proc, err := rt.Find(ctx, session.ID)
	if err != nil {
		// Process not found, mark session as stopped
		session.Status = StatusStopped
		if session.StoppedAt == nil {
			now := time.Now()
			session.StoppedAt = &now
		}
		// Update in memory
		m.mu.Lock()
		m.sessions[session.ID] = session
		m.mu.Unlock()
		// Save to disk
		_ = m.store.Save(ctx, session)
		return
	}

	// Update process state
	state := proc.State()
	switch state {
	case runtime.StateStopped:
		session.Status = StatusStopped
		if session.StoppedAt == nil {
			now := time.Now()
			session.StoppedAt = &now
		}
		// Try to get exit code
		if metadata := proc.Metadata(); metadata != nil {
			if metaMap := metadata.ToMap(); metaMap != nil {
				if exitCode, ok := metaMap["exit_code"].(int); ok {
					session.ExitCode = &exitCode
				}
			}
		}
	case runtime.StateFailed:
		session.Status = StatusFailed
		if session.StoppedAt == nil {
			now := time.Now()
			session.StoppedAt = &now
		}
		// Try to get exit code
		if metadata := proc.Metadata(); metadata != nil {
			if metaMap := metadata.ToMap(); metaMap != nil {
				if exitCode, ok := metaMap["exit_code"].(int); ok {
					session.ExitCode = &exitCode
				}
			}
		}
	case runtime.StateRunning:
		// Still running, keep current status
		session.Status = StatusRunning
	case runtime.StateStarting:
		// Session is still starting, keep current status
		session.Status = StatusStarting
	case runtime.StateUnknown:
		// Cannot determine state, keep current status
		// This might happen if runtime doesn't support state tracking
	}

	// Update in memory and save if status changed
	if state != runtime.StateRunning {
		m.mu.Lock()
		m.sessions[session.ID] = session
		m.mu.Unlock()
		_ = m.store.Save(ctx, session)
	}
}

// generateRandomSuffix generates a random 8-character hex string
func generateRandomSuffix() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp if random fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
