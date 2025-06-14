package session

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/idmap"
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
	now := time.Now()
	info := &Info{
		ID:          "test-mock-session",
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		StatusState: StatusState{
			Status:          StatusCreated,
			StatusChangedAt: now,
			LastOutputTime:  now,
		},
		Command: "echo 'Test session started'",
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
		CreatedAt: now,
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
	output, err := session.GetOutput(0)
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

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	manager := NewManager(store, wsManager, idMapper)

	// Replace tmux adapter with mock
	mockAdapter := tmux.NewMockAdapter()
	manager.tmuxAdapter = mockAdapter

	// Create session
	opts := Options{
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

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	manager := NewManager(store, wsManager, idMapper)

	// Replace with unavailable mock
	mockAdapter := tmux.NewMockAdapter()
	mockAdapter.SetAvailable(false)
	manager.tmuxAdapter = mockAdapter

	// Create session
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
	}

	_, err = manager.CreateSession(opts)
	if err == nil {
		t.Fatal("Expected error when creating session with unavailable tmux")
	}

	// Should return ErrTmuxNotAvailable
	if _, ok := err.(ErrTmuxNotAvailable); !ok {
		t.Errorf("Expected ErrTmuxNotAvailable, got %T: %v", err, err)
	}
}

func TestSessionStatus_MockAdapter(t *testing.T) {
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name: "test-workspace-status-mock",
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
	now := time.Now()
	info := &Info{
		ID:          "test-status-mock-session",
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		StatusState: StatusState{
			Status:          StatusCreated,
			StatusChangedAt: now,
			LastOutputTime:  now,
		},
		Command:   "test-command",
		CreatedAt: now,
	}

	// Save info
	if err := store.Save(info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create tmux session with mock adapter
	session := NewTmuxSession(info, store, mockAdapter, ws).(*tmuxSessionImpl)

	// Initialize the session as if it started
	session.info.StatusState.Status = StatusWorking
	session.info.TmuxSession = "test-session"

	// Create the session in the mock adapter
	err = mockAdapter.CreateSession("test-session", ws.Path)
	if err != nil {
		t.Fatalf("Failed to create mock session: %v", err)
	}
	mockAdapter.SetPaneContent("test-session", "initial output")

	// Test status behavior
	tests := []struct {
		name                 string
		setupFunc            func()
		expectedStatus       Status
		checkStatusChangedAt bool
	}{
		{
			name: "Initial working status remains working",
			setupFunc: func() {
				// First update to establish baseline
				session.UpdateStatus()
			},
			expectedStatus:       StatusWorking,
			checkStatusChangedAt: false,
		},
		{
			name: "Status remains working with new output",
			setupFunc: func() {
				// Change output
				mockAdapter.SetPaneContent("test-session", "new output")
				session.UpdateStatus()
			},
			expectedStatus:       StatusWorking,
			checkStatusChangedAt: false,
		},
		{
			name: "Status remains working within idle threshold",
			setupFunc: func() {
				// Wait a bit but less than idle threshold
				time.Sleep(1 * time.Second)
				session.UpdateStatus()
			},
			expectedStatus:       StatusWorking,
			checkStatusChangedAt: false,
		},
		{
			name: "Status becomes idle after no output for idle threshold",
			setupFunc: func() {
				// Reset to a known state
				mockAdapter.SetPaneContent("test-session", "idle test output")

				// Reset the lastStatusCheck to ensure cache doesn't interfere
				session.lastStatusCheck = time.Time{}

				// First ensure we have current state by calling UpdateStatus
				// This will capture the current output and set lastOutputContent
				err := session.UpdateStatus()
				if err != nil {
					t.Fatalf("First UpdateStatus failed: %v", err)
				}

				// Wait for idle threshold to pass
				time.Sleep(3500 * time.Millisecond) // Well over 3 seconds

				// Reset the lastStatusCheck again to ensure second update runs
				session.lastStatusCheck = time.Time{}

				// Update status again - should detect idle since output hasn't changed
				err = session.UpdateStatus()
				if err != nil {
					t.Fatalf("Second UpdateStatus failed: %v", err)
				}
			},
			expectedStatus:       StatusIdle,
			checkStatusChangedAt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc()
			status := session.Status()
			if status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, status)
			}

			// Check StatusChangedAt is set correctly
			if tt.checkStatusChangedAt {
				if session.info.StatusState.StatusChangedAt.IsZero() {
					t.Error("Expected StatusChangedAt to be set when status changes")
				}
			}
		})
	}
}
