// Package session provides session management for amux
package session

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/task"
)

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
	WorkspaceID string                 `json:"workspace_id" yaml:"workspace_id"`
	TaskName    string                 `json:"task_name" yaml:"task_name"`
	Runtime     string                 `json:"runtime" yaml:"runtime"`
	Status      Status                 `json:"status" yaml:"status"`
	ProcessID   string                 `json:"process_id,omitempty" yaml:"process_id,omitempty"`
	StartedAt   time.Time              `json:"started_at" yaml:"started_at"`
	StoppedAt   *time.Time             `json:"stopped_at,omitempty" yaml:"stopped_at,omitempty"`
	ExitCode    *int                   `json:"exit_code,omitempty" yaml:"exit_code,omitempty"`
	Command     []string               `json:"command" yaml:"command"`
	Environment map[string]string      `json:"environment,omitempty" yaml:"environment,omitempty"`
	WorkingDir  string                 `json:"working_dir" yaml:"working_dir"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Runtime process reference (not serialized)
	process runtime.Process `json:"-" yaml:"-"`
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
}

// CreateOptions defines options for creating a session
type CreateOptions struct {
	WorkspaceID    string                 // Workspace to run in
	TaskName       string                 // Task to execute (optional)
	Command        []string               // Direct command (if no task)
	Runtime        string                 // Runtime to use (default: local)
	Environment    map[string]string      // Additional environment variables
	WorkingDir     string                 // Working directory override
	Metadata       map[string]interface{} // Additional metadata
	RuntimeOptions runtime.RuntimeOptions // Runtime-specific options
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
	mu        sync.RWMutex
	sessions  map[string]*Session
	store     Store
	runtimes  map[string]runtime.Runtime
	tasks     *task.Manager
	idCounter int
}

// NewManager creates a new session manager
func NewManager(store Store, runtimes map[string]runtime.Runtime, tasks *task.Manager) Manager {
	return &manager{
		sessions:  make(map[string]*Session),
		store:     store,
		runtimes:  runtimes,
		tasks:     tasks,
		idCounter: 0,
	}
}

// Create starts a new session
func (m *manager) Create(ctx context.Context, opts CreateOptions) (*Session, error) {
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
		WorkingDir:  opts.WorkingDir,
		Environment: opts.Environment,
		Options:     opts.RuntimeOptions,
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

	// Start the process
	process, err := rt.Execute(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to execute: %w", err)
	}

	// Create session
	m.mu.Lock()
	sessionID := fmt.Sprintf("session-%d", m.idCounter+1)
	m.idCounter++
	m.mu.Unlock()

	// Get process metadata
	var metadata map[string]interface{}
	if processMetadata := process.Metadata(); processMetadata != nil {
		metadata = processMetadata.ToMap()
		// Merge with any user-provided metadata
		if opts.Metadata != nil {
			for k, v := range opts.Metadata {
				metadata[k] = v
			}
		}
	} else {
		metadata = opts.Metadata
	}

	session := &Session{
		ID:          sessionID,
		WorkspaceID: opts.WorkspaceID,
		TaskName:    opts.TaskName,
		Runtime:     opts.Runtime,
		Status:      StatusRunning,
		ProcessID:   process.ID(),
		StartedAt:   process.StartTime(),
		Command:     spec.Command,
		Environment: spec.Environment,
		WorkingDir:  spec.WorkingDir,
		Metadata:    metadata,
		process:     process,
	}

	// Store session
	m.mu.Lock()
	m.sessions[session.ID] = session
	m.mu.Unlock()

	// Save to persistent store
	if err := m.store.Save(ctx, session); err != nil {
		// Try to clean up
		_ = process.Kill(ctx)
		m.mu.Lock()
		delete(m.sessions, session.ID)
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Start monitoring goroutine
	go m.monitorSession(ctx, session)

	// Start log capture if process supports it
	go m.captureSessionLogs(ctx, session)

	return session, nil
}

// monitorSession monitors a session for state changes
func (m *manager) monitorSession(ctx context.Context, session *Session) {
	// Wait for process to complete
	_ = session.process.Wait(ctx)

	// Update session state
	state := session.process.State()
	var status Status
	switch state {
	case runtime.StateStarting:
		status = StatusStarting
	case runtime.StateRunning:
		status = StatusRunning
	case runtime.StateStopped:
		status = StatusStopped
	case runtime.StateFailed:
		status = StatusFailed
	case runtime.StateUnknown:
		status = StatusUnknown
	default:
		status = StatusUnknown
	}

	// Get exit code
	exitCode, _ := session.process.ExitCode()
	now := time.Now()

	// Update session
	m.mu.Lock()
	session.Status = status
	session.StoppedAt = &now
	if exitCode >= 0 {
		session.ExitCode = &exitCode
	}
	m.mu.Unlock()

	// Save to store
	_ = m.store.Save(ctx, session)
}

// Get retrieves a session by ID
func (m *manager) Get(ctx context.Context, id string) (*Session, error) {
	m.mu.RLock()
	session, ok := m.sessions[id]
	m.mu.RUnlock()

	if ok {
		return session, nil
	}

	// Try to load from store
	session, err := m.store.Load(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	// Try to reconnect to process if still running
	if session.Status == StatusRunning && session.ProcessID != "" {
		rt, ok := m.runtimes[session.Runtime]
		if ok {
			if process, err := rt.Find(ctx, session.ProcessID); err == nil {
				session.process = process
				m.mu.Lock()
				m.sessions[session.ID] = session
				m.mu.Unlock()
				go m.monitorSession(ctx, session)
			}
		}
	}

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
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessionMap := make(map[string]*Session)
	for _, s := range sessions {
		sessionMap[s.ID] = s
	}

	// Override with in-memory sessions (more up-to-date)
	for _, s := range m.sessions {
		if workspaceID == "" || s.WorkspaceID == workspaceID {
			sessionMap[s.ID] = s
		}
	}

	// Convert back to slice
	result := make([]*Session, 0, len(sessionMap))
	for _, s := range sessionMap {
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

	if session.process == nil {
		return fmt.Errorf("session process not available")
	}

	if err := session.process.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop session: %w", err)
	}

	return nil
}

// Kill forcefully terminates a session
func (m *manager) Kill(ctx context.Context, id string) error {
	session, err := m.Get(ctx, id)
	if err != nil {
		return err
	}

	if session.process == nil {
		return fmt.Errorf("session process not available")
	}

	if err := session.process.Kill(ctx); err != nil {
		return fmt.Errorf("failed to kill session: %w", err)
	}

	return nil
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

	// For tmux runtime, we need special handling
	if session.Runtime == "tmux" {
		// Use tmux attach-session command
		cmd := exec.Command("tmux", "attach-session", "-t", session.ProcessID)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to attach to tmux session: %w", err)
		}
		return nil
	}

	return fmt.Errorf("attach not supported for runtime: %s", session.Runtime)
}

// Logs returns the logs for a session
func (m *manager) Logs(ctx context.Context, id string, follow bool) (LogReader, error) {
	session, err := m.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if session.process == nil {
		// Try to get logs from storage
		return m.store.GetLogs(ctx, id)
	}

	// Get process output
	stdout, _ := session.process.Output()

	// TODO: Implement proper log reader that combines stdout/stderr
	// For now, return stdout
	return &simpleLogReader{reader: stdout}, nil
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

// captureSessionLogs captures session logs to storage
func (m *manager) captureSessionLogs(ctx context.Context, session *Session) {
	if session.process == nil {
		return
	}

	// Get process output
	stdout, stderr := session.process.Output()

	// Create a multi-reader to capture both stdout and stderr
	pr, pw := io.Pipe()

	// Copy both streams to the pipe
	go func() {
		defer func() {
			_ = pw.Close()
		}()

		// Use io.MultiWriter to write to both pipe and capture
		if stdout != nil {
			_, _ = io.Copy(pw, stdout)
		}
		if stderr != nil {
			_, _ = io.Copy(pw, stderr)
		}
	}()

	// Save logs to store
	if fileStore, ok := m.store.(*FileStore); ok {
		_ = fileStore.SaveLogs(ctx, session.ID, pr)
	}
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
