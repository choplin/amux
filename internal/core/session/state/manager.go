package state

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ChangeHandler is called when state transitions occur
type ChangeHandler func(ctx context.Context, from, to Status, sessionID, workspaceID string) error

// Manager manages session state transitions
type Manager struct {
	sessionID     string
	workspaceID   string
	stateFilePath string
	handlers      []ChangeHandler
	logger        *slog.Logger
	mu            sync.Mutex
}

// Data represents the persisted state information
type Data struct {
	Status          Status    `json:"status"`
	StatusChangedAt time.Time `json:"status_changed_at"`
}

// NewManager creates a new state manager
func NewManager(sessionID, workspaceID, stateDir string, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		sessionID:     sessionID,
		workspaceID:   workspaceID,
		stateFilePath: filepath.Join(stateDir, "state.json"),
		handlers:      make([]ChangeHandler, 0),
		logger:        logger,
	}
}

// AddChangeHandler adds a handler to be called on state transitions
func (m *Manager) AddChangeHandler(handler ChangeHandler) {
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

// GetData returns the full state data
func (m *Manager) GetData() (*Data, error) {
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
		data = &Data{
			Status:          StatusCreated,
			StatusChangedAt: time.Now(),
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
	m.logger.Debug("state transition",
		"session", m.sessionID,
		"from", currentStatus,
		"to", newStatus)

	// Call handlers before transition
	for _, handler := range m.handlers {
		if err := handler(ctx, currentStatus, newStatus, m.sessionID, m.workspaceID); err != nil {
			m.logger.Error("state change handler failed",
				"session", m.sessionID,
				"from", currentStatus,
				"to", newStatus,
				"error", err)
			return fmt.Errorf("state change handler failed: %w", err)
		}
	}

	// Update state
	now := time.Now()
	data.Status = newStatus
	data.StatusChangedAt = now

	// Persist state
	if err := m.saveState(data); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	m.logger.Info("state transition completed",
		"session", m.sessionID,
		"from", currentStatus,
		"to", newStatus)

	return nil
}

// loadState loads state from file
func (m *Manager) loadState() (*Data, error) {
	file, err := os.Open(m.stateFilePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	var data Data
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}

	return &data, nil
}

// saveState saves state to file
func (m *Manager) saveState(data *Data) error {
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
		_ = file.Close()
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to encode state: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to close file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, m.stateFilePath); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}
