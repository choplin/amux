package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger interface for state manager logging
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// StateChangeHandler is called when state transitions occur
type StateChangeHandler func(ctx context.Context, from, to Status, sessionID, workspaceID string) error

// Manager manages session state transitions
type Manager struct {
	sessionID     string
	workspaceID   string
	stateFilePath string
	handlers      []StateChangeHandler
	logger        Logger
	mu            sync.Mutex
}

// StateData represents the persisted state information
type StateData struct {
	Status           Status    `json:"status"`
	StatusChangedAt  time.Time `json:"status_changed_at"`
	LastActivityTime time.Time `json:"last_activity_time"`
}

// NewManager creates a new state manager
func NewManager(sessionID, workspaceID, stateDir string, logger Logger) *Manager {
	return &Manager{
		sessionID:     sessionID,
		workspaceID:   workspaceID,
		stateFilePath: filepath.Join(stateDir, "state.json"),
		handlers:      make([]StateChangeHandler, 0),
		logger:        logger,
	}
}

// AddStateChangeHandler adds a handler to be called on state transitions
func (m *Manager) AddStateChangeHandler(handler StateChangeHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

// CurrentState returns the current state
func (m *Manager) CurrentState() (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := m.loadState()
	if err != nil {
		if os.IsNotExist(err) {
			// Default to created state if file doesn't exist
			return StatusCreated, nil
		}
		return "", fmt.Errorf("failed to load state: %w", err)
	}

	return data.Status, nil
}

// StateData returns the full state data
func (m *Manager) StateData() (*StateData, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.loadState()
}

// TransitionTo attempts to transition to the specified state
func (m *Manager) TransitionTo(ctx context.Context, newStatus Status) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load current state
	data, err := m.loadState()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load state: %w", err)
	}

	// Default state if file doesn't exist
	if data == nil {
		data = &StateData{
			Status:           StatusCreated,
			StatusChangedAt:  time.Now(),
			LastActivityTime: time.Now(),
		}
	}

	currentStatus := data.Status

	// Check if already in target state
	if currentStatus == newStatus {
		return &ErrAlreadyInState{State: currentStatus}
	}

	// Validate transition
	if err := ValidateTransition(currentStatus, newStatus); err != nil {
		return err
	}

	// Log transition
	if m.logger != nil {
		m.logger.Debug("state transition",
			"session", m.sessionID,
			"from", currentStatus,
			"to", newStatus)
	}

	// Call handlers before transition
	for _, handler := range m.handlers {
		if err := handler(ctx, currentStatus, newStatus, m.sessionID, m.workspaceID); err != nil {
			if m.logger != nil {
				m.logger.Error("state change handler failed",
					"session", m.sessionID,
					"from", currentStatus,
					"to", newStatus,
					"error", err)
			}
			return fmt.Errorf("state change handler failed: %w", err)
		}
	}

	// Update state
	now := time.Now()
	data.Status = newStatus
	data.StatusChangedAt = now
	data.LastActivityTime = now

	// Persist state
	if err := m.saveState(data); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	if m.logger != nil {
		m.logger.Info("state transition completed",
			"session", m.sessionID,
			"from", currentStatus,
			"to", newStatus)
	}

	return nil
}

// UpdateActivity updates the last activity time without changing state
func (m *Manager) UpdateActivity() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := m.loadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	data.LastActivityTime = time.Now()

	return m.saveState(data)
}

// loadState loads state from file
func (m *Manager) loadState() (*StateData, error) {
	file, err := os.Open(m.stateFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data StateData
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}

	return &data, nil
}

// saveState saves state to file
func (m *Manager) saveState(data *StateData) error {
	// Ensure directory exists
	dir := filepath.Dir(m.stateFilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Write to temporary file first
	tmpFile := m.stateFilePath + ".tmp"
	file, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		file.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("failed to encode state: %w", err)
	}

	if err := file.Close(); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to close file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, m.stateFilePath); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}
