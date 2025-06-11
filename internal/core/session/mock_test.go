package session

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/workspace"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestTmuxSession_WithMock(t *testing.T) {
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name: "test-workspace",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create store
	store, err := NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create mock adapter
	mockAdapter := tmux.NewMockAdapter()

	// Create session info
	info := &SessionInfo{
		ID:          "test-mock-session",
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Status:      StatusCreated,
		Command:     "echo 'Test session started'",
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
		CreatedAt: time.Now(),
	}

	// Save info
	if err := store.Save(info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create tmux session with mock
	session := NewTmuxSession(info, store, mockAdapter, ws)

	// Start session
	ctx := context.Background()
	if err := session.Start(ctx); err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Verify session was created in mock
	sessions := mockAdapter.GetSessions()
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session in mock, got %d", len(sessions))
	}

	// Verify environment was set
	for name := range sessions {
		env := mockAdapter.GetSessionEnvironment(name)
		if env["TEST_VAR"] != "test_value" {
			t.Errorf("Expected TEST_VAR=test_value in session %s", name)
		}
		if env["AMUX_WORKSPACE_ID"] != ws.ID {
			t.Errorf("Expected AMUX_WORKSPACE_ID=%s in session %s", ws.ID, name)
		}
	}

	// Send input
	if err := session.SendInput("echo 'Hello from tmux'"); err != nil {
		t.Errorf("Failed to send input: %v", err)
	}

	// Get output
	output, err := session.GetOutput()
	if err != nil {
		t.Errorf("Failed to get output: %v", err)
	}

	// Should contain both the initial command and our test command
	outputStr := string(output)
	if !contains(outputStr, "echo 'Test session started'") {
		t.Errorf("Output should contain initial command")
	}
	if !contains(outputStr, "Hello from tmux") {
		t.Errorf("Output should contain 'Hello from tmux'")
	}

	// Stop session
	if err := session.Stop(); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	// Verify session was killed
	if mockAdapter.SessionExists(info.TmuxSession) {
		t.Error("Session should not exist after stop")
	}
}

func TestManager_WithMockAdapter(t *testing.T) {
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name: "test-workspace",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create store and manager
	store, err := NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	manager := NewManager(store, wsManager, nil)

	// Replace tmux adapter with mock
	mockAdapter := tmux.NewMockAdapter()
	manager.tmuxAdapter = mockAdapter

	// Create session
	opts := SessionOptions{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     "test command",
		Environment: map[string]string{
			"API_KEY": "secret",
		},
	}

	session, err := manager.CreateSession(opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Should create a tmux-backed session
	if _, ok := session.(*tmuxSessionImpl); !ok {
		t.Error("Expected tmux-backed session")
	}

	// Start the session
	ctx := context.Background()
	if err := session.Start(ctx); err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Verify mock state
	if len(mockAdapter.GetSessions()) != 1 {
		t.Errorf("Expected 1 session in mock adapter")
	}
}

func TestManager_WithUnavailableTmux(t *testing.T) {
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name: "test-workspace",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create store and manager
	store, err := NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	manager := NewManager(store, wsManager, nil)

	// Replace with unavailable mock
	mockAdapter := tmux.NewMockAdapter()
	mockAdapter.SetAvailable(false)
	manager.tmuxAdapter = mockAdapter

	// Create session
	opts := SessionOptions{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
	}

	session, err := manager.CreateSession(opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Should create a basic session when tmux unavailable
	if _, ok := session.(*sessionImpl); !ok {
		t.Error("Expected basic session when tmux unavailable")
	}

}
