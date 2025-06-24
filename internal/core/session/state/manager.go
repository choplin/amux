package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// Manager manages session state transitions with inter-process safety
type Manager struct {
	sessionID   string
	workspaceID string
	basePath    string // .amux/sessions/{id}/
	logger      Logger
	mu          sync.Mutex

	// Callbacks for state changes
	onStateChange []ChangeHandler
}

// ChangeHandler is called when state transitions occur
type ChangeHandler func(ctx context.Context, from, to Status, sessionID, workspaceID string) error

// NewManager creates a new state manager for a session
func NewManager(sessionID, workspaceID, basePath string, logger Logger) *Manager {
	if logger == nil {
		logger = &noopLogger{}
	}
	return &Manager{
		sessionID:   sessionID,
		workspaceID: workspaceID,
		basePath:    basePath,
		logger:      logger,
	}
}

// AddStateChangeHandler registers a handler for state changes
func (m *Manager) AddStateChangeHandler(handler ChangeHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onStateChange = append(m.onStateChange, handler)
}

// GetState returns the current state with context
func (m *Manager) GetState(ctx context.Context) (*Data, error) {
	lock, err := m.acquireLock(ctx, ReadLock)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire read lock: %w", err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			m.logger.Debug("failed to release read lock", "error", err)
		}
	}()

	state, err := m.loadState()
	if err != nil {
		if os.IsNotExist(err) {
			// Return default state if file doesn't exist
			return &Data{
				State:       StatusCreated,
				SessionID:   m.sessionID,
				WorkspaceID: m.workspaceID,
				UpdatedAt:   time.Now(),
			}, nil
		}
		return nil, err
	}
	return state, nil
}

// UpdateActivity updates activity tracking data
func (m *Manager) UpdateActivity(ctx context.Context, outputHash uint32, outputTime time.Time) error {
	lock, err := m.acquireLock(ctx, WriteLock)
	if err != nil {
		return fmt.Errorf("failed to acquire write lock: %w", err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			m.logger.Debug("failed to release write lock", "error", err)
		}
	}()

	// Load current state
	current, err := m.loadState()
	if err != nil {
		return fmt.Errorf("failed to load current state: %w", err)
	}

	// Update activity fields
	current.LastOutputHash = outputHash
	current.LastOutputTime = outputTime
	current.LastStatusCheck = time.Now()

	// Save state
	if err := m.saveState(current); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// TransitionTo transitions to a new state with validation and actions
func (m *Manager) TransitionTo(ctx context.Context, newState Status) error {
	lock, err := m.acquireLock(ctx, WriteLock)
	if err != nil {
		return fmt.Errorf("failed to acquire write lock: %w", err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			m.logger.Debug("failed to release write lock", "error", err)
		}
	}()

	// Load current state
	current, err := m.loadState()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load current state: %w", err)
	}

	// Handle initial state
	if current == nil {
		current = &Data{
			State:       StatusCreated,
			SessionID:   m.sessionID,
			WorkspaceID: m.workspaceID,
		}
	}

	// Validate transition
	if !isValidTransition(current.State, newState) {
		return &ErrInvalidTransition{From: current.State, To: newState}
	}

	// Execute state change handlers
	for _, handler := range m.onStateChange {
		if err := handler(ctx, current.State, newState, m.sessionID, m.workspaceID); err != nil {
			m.logger.Error("state change handler failed",
				"from", current.State,
				"to", newState,
				"error", err)
			// Don't fail the transition, just log
		}
	}

	// Update state atomically
	newData := &Data{
		State:       newState,
		UpdatedAt:   time.Now(),
		UpdatedBy:   os.Getpid(),
		SessionID:   m.sessionID,
		WorkspaceID: m.workspaceID,

		// Preserve activity tracking data
		LastOutputHash:  current.LastOutputHash,
		LastOutputTime:  current.LastOutputTime,
		LastStatusCheck: current.LastStatusCheck,
	}

	if err := m.saveState(newData); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	m.logger.Info("state transitioned",
		"session", m.sessionID,
		"from", current.State,
		"to", newState)

	return nil
}

// acquireLock acquires a file-based lock with timeout
func (m *Manager) acquireLock(ctx context.Context, lockType LockType) (Lock, error) {
	lockPath := filepath.Join(m.basePath, fmt.Sprintf(".lock.%s", lockType))

	// Ensure directory exists
	if err := os.MkdirAll(m.basePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Try to acquire lock with timeout
	deadline := time.Now().Add(5 * time.Second)
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return nil, fmt.Errorf("failed to open lock file: %w", err)
		}

		// Try to acquire flock
		lockMode := syscall.LOCK_SH
		if lockType == WriteLock {
			lockMode = syscall.LOCK_EX
		}

		err = syscall.Flock(int(file.Fd()), lockMode|syscall.LOCK_NB)
		if err == nil {
			// Got the lock
			lock := &sessionLock{
				file: file,
				mode: lockType,
				path: lockPath,
			}

			if m.logger != nil {
				m.logger.Debug("acquired state lock",
					"session", m.sessionID,
					"type", lockType)
			}

			// Write lock info for debugging
			if lockType == WriteLock {
				info := &LockInfo{
					PID:        os.Getpid(),
					Operation:  getCallerOperation(),
					AcquiredAt: time.Now(),
				}
				infoPath := lockPath + ".info"
				if err := writeJSON(infoPath, info); err != nil {
					m.logger.Warn("failed to write lock info", "error", err)
				}
			}

			return lock, nil
		}

		// Check if we should retry
		if err != syscall.EWOULDBLOCK || time.Now().After(deadline) {
			if closeErr := file.Close(); closeErr != nil {
				return nil, closeErr
			}

			// Try to get lock holder info
			var lockInfo *LockInfo
			infoPath := lockPath + ".info"
			if data, err := os.ReadFile(infoPath); err == nil {
				if unmarshalErr := json.Unmarshal(data, &lockInfo); unmarshalErr != nil {
					// Ignore unmarshal errors, lockInfo will remain nil
					lockInfo = nil
				}
			}

			return nil, &ErrSessionLocked{
				SessionID: m.sessionID,
				LockedBy:  lockInfo,
			}
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			if closeErr := file.Close(); closeErr != nil {
				return nil, closeErr
			}
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
			// Retry
		}
	}
}

// loadState loads the current state from disk
func (m *Manager) loadState() (*Data, error) {
	statePath := filepath.Join(m.basePath, "state.json")

	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}

	var state Data
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// saveState saves the state atomically
func (m *Manager) saveState(state *Data) error {
	statePath := filepath.Join(m.basePath, "state.json")
	tmpPath := statePath + ".tmp"

	// Marshal state
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, statePath); err != nil {
		if removeErr := os.Remove(tmpPath); removeErr != nil && !os.IsNotExist(removeErr) {
			// Log error but continue
			return fmt.Errorf("failed to rename state file: %w (cleanup error: %v)", err, removeErr)
		}
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// sessionLock represents a file-based lock for session operations
type sessionLock struct {
	file *os.File
	mode LockType
	path string
}

// Release releases the lock
func (l *sessionLock) Release() error {
	if l.file == nil {
		return nil
	}

	// Remove lock info file for write locks
	if l.mode == WriteLock {
		if err := os.Remove(l.path + ".info"); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	// Release flock and close file
	if err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN); err != nil {
		return err
	}
	return l.file.Close()
}

// isValidTransition checks if a state transition is allowed
func isValidTransition(from, to Status) bool {
	validTransitions := map[Status][]Status{
		StatusCreated:  {StatusStarting, StatusFailed, StatusOrphaned},
		StatusStarting: {StatusRunning, StatusWorking, StatusFailed, StatusOrphaned},
		StatusRunning:  {StatusWorking, StatusIdle, StatusStopping, StatusFailed, StatusCompleted, StatusOrphaned},
		StatusWorking:  {StatusIdle, StatusRunning, StatusStopping, StatusFailed, StatusCompleted, StatusOrphaned},
		StatusIdle:     {StatusWorking, StatusRunning, StatusStopping, StatusFailed, StatusCompleted, StatusOrphaned},
		StatusStopping: {StatusStopped, StatusFailed},
		// Terminal states cannot transition
		StatusStopped:   {},
		StatusFailed:    {},
		StatusCompleted: {},
		StatusOrphaned:  {},
	}

	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}

	for _, s := range allowed {
		if s == to {
			return true
		}
	}

	return false
}

// writeJSON writes data as JSON to a file
func writeJSON(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0o600)
}

// getCallerOperation attempts to determine what operation is being performed
func getCallerOperation() string {
	// This is a simple implementation
	// In production, might use runtime.Caller to get more info
	return "session-operation"
}

// noopLogger is a no-op logger implementation
type noopLogger struct{}

func (n *noopLogger) Debug(msg string, args ...interface{}) {}
func (n *noopLogger) Info(msg string, args ...interface{})  {}
func (n *noopLogger) Warn(msg string, args ...interface{})  {}
func (n *noopLogger) Error(msg string, args ...interface{}) {}

// ErrInvalidTransition is returned when an invalid state transition is attempted
type ErrInvalidTransition struct {
	From Status
	To   Status
}

func (e *ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid transition from %s to %s", e.From, e.To)
}
