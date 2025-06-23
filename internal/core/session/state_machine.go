package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Logger interface for state machine logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// LockMode represents the type of lock to acquire
type LockMode int

const (
	// ReadLock allows multiple concurrent readers
	ReadLock LockMode = iota
	// WriteLock allows only a single writer
	WriteLock
)

// StateData represents the persistent state information
type StateData struct {
	State       Status    `json:"state"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   int       `json:"updated_by"` // PID
	SessionID   string    `json:"session_id"`
	WorkspaceID string    `json:"workspace_id"`
}

// LockInfo contains information about who holds a lock
type LockInfo struct {
	PID       int       `json:"pid"`
	Operation string    `json:"operation"`
	Timestamp time.Time `json:"timestamp"`
}

// SessionLock represents a file-based lock for session operations
type SessionLock struct {
	file *os.File
	mode LockMode
	path string
}

// Release releases the lock
func (l *SessionLock) Release() error {
	if l.file == nil {
		return nil
	}

	// Remove lock info file for write locks
	if l.mode == WriteLock {
		_ = os.Remove(l.path + ".info")
	}

	// Release flock and close file
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	return l.file.Close()
}

// StateMachine manages session state transitions with inter-process safety
type StateMachine struct {
	sessionID   string
	workspaceID string
	basePath    string // .amux/sessions/{id}/

	// Callbacks for state changes
	onStateChange []StateChangeHandler

	// Logger
	logger Logger
}

// StateChangeHandler is called when state transitions occur
type StateChangeHandler func(ctx context.Context, from, to Status, sessionID, workspaceID string) error

// NewStateMachine creates a new state machine for a session
func NewStateMachine(sessionID, workspaceID, basePath string, logger Logger) *StateMachine {
	return &StateMachine{
		sessionID:   sessionID,
		workspaceID: workspaceID,
		basePath:    basePath,
		logger:      logger,
	}
}

// AddStateChangeHandler registers a handler for state changes
func (sm *StateMachine) AddStateChangeHandler(handler StateChangeHandler) {
	sm.onStateChange = append(sm.onStateChange, handler)
}

// GetCurrentState returns the current state with read lock
func (sm *StateMachine) GetCurrentState(ctx context.Context) (*StateData, error) {
	lock, err := sm.acquireLock(ctx, ReadLock)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire read lock: %w", err)
	}
	defer lock.Release()

	return sm.loadState()
}

// TransitionTo transitions to a new state with validation and actions
func (sm *StateMachine) TransitionTo(ctx context.Context, newState Status) error {
	lock, err := sm.acquireLock(ctx, WriteLock)
	if err != nil {
		return fmt.Errorf("failed to acquire write lock: %w", err)
	}
	defer lock.Release()

	// Load current state
	current, err := sm.loadState()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load current state: %w", err)
	}

	// Handle initial state
	if current == nil {
		current = &StateData{
			State:       StatusCreated,
			SessionID:   sm.sessionID,
			WorkspaceID: sm.workspaceID,
		}
	}

	// Validate transition
	if !isValidTransition(current.State, newState) {
		return &ErrInvalidTransition{From: current.State, To: newState}
	}

	// Execute state change handlers
	for _, handler := range sm.onStateChange {
		if err := handler(ctx, current.State, newState, sm.sessionID, sm.workspaceID); err != nil {
			sm.logger.Error("state change handler failed",
				"from", current.State,
				"to", newState,
				"error", err)
			// Don't fail the transition, just log
		}
	}

	// Update state atomically
	newData := &StateData{
		State:       newState,
		UpdatedAt:   time.Now(),
		UpdatedBy:   os.Getpid(),
		SessionID:   sm.sessionID,
		WorkspaceID: sm.workspaceID,
	}

	if err := sm.saveState(newData); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	sm.logger.Info("state transitioned",
		"session", sm.sessionID,
		"from", current.State,
		"to", newState)

	return nil
}

// acquireLock acquires a file-based lock with timeout
func (sm *StateMachine) acquireLock(ctx context.Context, mode LockMode) (*SessionLock, error) {
	lockPath := filepath.Join(sm.basePath, "session.lock")

	// Ensure directory exists
	if err := os.MkdirAll(sm.basePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	// Open or create lock file
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Determine lock flags
	lockFlag := syscall.LOCK_SH
	if mode == WriteLock {
		lockFlag = syscall.LOCK_EX
	}

	// Try to acquire lock with timeout
	deadline := time.Now().Add(5 * time.Second)

	for {
		err := syscall.Flock(int(file.Fd()), lockFlag|syscall.LOCK_NB)
		if err == nil {
			// Successfully acquired lock
			lock := &SessionLock{
				file: file,
				mode: mode,
				path: lockPath,
			}

			// Write lock info for write locks
			if mode == WriteLock {
				info := &LockInfo{
					PID:       os.Getpid(),
					Operation: getCallerOperation(),
					Timestamp: time.Now(),
				}
				infoPath := lockPath + ".info"
				if err := writeJSON(infoPath, info); err != nil {
					sm.logger.Warn("failed to write lock info", "error", err)
				}
			}

			return lock, nil
		}

		// Check if we should retry
		if err != syscall.EWOULDBLOCK || time.Now().After(deadline) {
			file.Close()

			// Try to get lock holder info
			var lockInfo *LockInfo
			infoPath := lockPath + ".info"
			if data, err := os.ReadFile(infoPath); err == nil {
				_ = json.Unmarshal(data, &lockInfo)
			}

			return nil, &ErrSessionLocked{
				SessionID: sm.sessionID,
				LockedBy:  lockInfo,
			}
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			file.Close()
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
			// Retry
		}
	}
}

// loadState loads the current state from disk
func (sm *StateMachine) loadState() (*StateData, error) {
	statePath := filepath.Join(sm.basePath, "state.json")

	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, err
	}

	var state StateData
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// saveState saves the state atomically
func (sm *StateMachine) saveState(state *StateData) error {
	statePath := filepath.Join(sm.basePath, "state.json")
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
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// isValidTransition checks if a state transition is allowed
func isValidTransition(from, to Status) bool {
	validTransitions := map[Status][]Status{
		StatusCreated:  {StatusStarting, StatusFailed},
		StatusStarting: {StatusRunning, StatusFailed},
		StatusRunning:  {StatusStopping, StatusFailed, StatusCompleted},
		StatusStopping: {StatusStopped, StatusFailed},
		// Terminal states cannot transition
		StatusStopped:   {},
		StatusFailed:    {},
		StatusCompleted: {},
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

// ErrInvalidTransition is returned when an invalid state transition is attempted
type ErrInvalidTransition struct {
	From Status
	To   Status
}

func (e *ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid transition from %s to %s", e.From, e.To)
}

// ErrSessionLocked is returned when a session is locked by another process
type ErrSessionLocked struct {
	SessionID string
	LockedBy  *LockInfo
}

func (e *ErrSessionLocked) Error() string {
	if e.LockedBy != nil {
		elapsed := time.Since(e.LockedBy.Timestamp)
		return fmt.Sprintf("session %s is locked by process %d (%s, %v ago)",
			e.SessionID, e.LockedBy.PID, e.LockedBy.Operation, elapsed.Round(time.Second))
	}
	return fmt.Sprintf("session %s is locked by another process", e.SessionID)
}
