package tmux

import (
	"fmt"
	"sync"
)

// MockAdapter is a mock implementation of tmux operations for testing
type MockAdapter struct {
	mu            sync.RWMutex
	sessions      map[string]*MockSession
	available     bool
	createError   error
	sendKeysError error
}

// MockSession represents a mock tmux session for testing
type MockSession struct {
	name        string
	workDir     string
	environment map[string]string
	output      []string
	pid         int
}

// NewMockAdapter creates a new mock adapter
func NewMockAdapter() *MockAdapter {
	return &MockAdapter{
		sessions:  make(map[string]*MockSession),
		available: true,
	}
}

// SetAvailable sets whether tmux is available
func (m *MockAdapter) SetAvailable(available bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.available = available
}

// SetCreateError sets an error to return from CreateSession
func (m *MockAdapter) SetCreateError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createError = err
}

// SetSendKeysError sets an error to return from SendKeys
func (m *MockAdapter) SetSendKeysError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendKeysError = err
}

// IsAvailable checks if tmux is available on the system
func (m *MockAdapter) IsAvailable() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.available
}

// CreateSession creates a new tmux session
func (m *MockAdapter) CreateSession(sessionName, workDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createError != nil {
		return m.createError
	}

	if _, exists := m.sessions[sessionName]; exists {
		return fmt.Errorf("session already exists: %s", sessionName)
	}

	m.sessions[sessionName] = &MockSession{
		name:        sessionName,
		workDir:     workDir,
		environment: make(map[string]string),
		output:      []string{},
		pid:         12345 + len(m.sessions),
	}

	return nil
}

// SessionExists checks if a tmux session exists
func (m *MockAdapter) SessionExists(sessionName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.sessions[sessionName]
	return exists
}

// KillSession kills a tmux session
func (m *MockAdapter) KillSession(sessionName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionName)
	return nil
}

// SendKeys sends keystrokes to a tmux session
func (m *MockAdapter) SendKeys(sessionName, keys string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendKeysError != nil {
		return m.sendKeysError
	}

	session, exists := m.sessions[sessionName]
	if !exists {
		return fmt.Errorf("session does not exist: %s", sessionName)
	}

	// Simulate command execution by adding to output
	session.output = append(session.output, keys)

	// Simulate some common command outputs
	switch keys {
	case "echo 'Hello from tmux'":
		session.output = append(session.output, "Hello from tmux")
	case "echo $TEST_VAR1":
		if val, ok := session.environment["TEST_VAR1"]; ok {
			session.output = append(session.output, val)
		}
	}

	return nil
}

// CapturePane captures the content of the current pane
func (m *MockAdapter) CapturePane(sessionName string) (string, error) {
	// Use CapturePaneWithOptions with 0 to capture all lines
	return m.CapturePaneWithOptions(sessionName, 0)
}

// AttachSession attaches to a tmux session
func (m *MockAdapter) AttachSession(sessionName string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.sessions[sessionName]; !exists {
		return fmt.Errorf("session does not exist: %s", sessionName)
	}

	return fmt.Errorf("to attach to session, run: tmux attach-session -t %s", sessionName)
}

// ListSessions returns a list of active tmux sessions
func (m *MockAdapter) ListSessions() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]string, 0, len(m.sessions))
	for name := range m.sessions {
		sessions = append(sessions, name)
	}

	return sessions, nil
}

// GetSessionPID gets the PID of the main process in a tmux session
func (m *MockAdapter) GetSessionPID(sessionName string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return 0, fmt.Errorf("session does not exist: %s", sessionName)
	}

	return session.pid, nil
}

// SetEnvironment sets environment variables in a tmux session
func (m *MockAdapter) SetEnvironment(sessionName string, env map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return fmt.Errorf("session does not exist: %s", sessionName)
	}

	// Copy environment variables
	for k, v := range env {
		session.environment[k] = v
	}

	return nil
}

// ResizeWindow resizes the tmux window to standard dimensions
func (m *MockAdapter) ResizeWindow(sessionName string, width, height int) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.sessions[sessionName]; !exists {
		return fmt.Errorf("session does not exist: %s", sessionName)
	}

	// Mock implementation - just return success
	return nil
}

// GetSessions returns the internal sessions map for testing
func (m *MockAdapter) GetSessions() map[string]*MockSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	sessions := make(map[string]*MockSession)
	for k, v := range m.sessions {
		sessions[k] = v
	}
	return sessions
}

// GetSessionEnvironment returns the environment for a specific session
func (m *MockAdapter) GetSessionEnvironment(sessionName string) map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if session, exists := m.sessions[sessionName]; exists {
		// Return a copy
		env := make(map[string]string)
		for k, v := range session.environment {
			env[k] = v
		}
		return env
	}
	return nil
}

// AppendSessionOutput appends output to a specific session (for testing streaming)
func (m *MockAdapter) AppendSessionOutput(sessionName string, output string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return fmt.Errorf("session does not exist: %s", sessionName)
	}

	session.output = append(session.output, output)
	return nil
}
// CapturePaneWithOptions captures the content with specified options
func (m *MockAdapter) CapturePaneWithOptions(sessionName string, lines int) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionName]
	if !exists {
		return "", fmt.Errorf("session does not exist: %s", sessionName)
	}

	// Return output joined by newlines
	output := ""
	outputLines := session.output

	// Limit to last N lines if specified
	if lines > 0 && len(outputLines) > lines {
		outputLines = outputLines[len(outputLines)-lines:]
	}

	for _, line := range outputLines {
		output += line + "\n"
	}

	return output, nil
}

// SetPaneContent sets the pane content for testing
func (m *MockAdapter) SetPaneContent(sessionName string, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, exists := m.sessions[sessionName]; exists {
		session.output = []string{content}
	}
}
