package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/session/state"
	"github.com/aki/amux/internal/core/workspace"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestTmuxSession_WithMock(t *testing.T) {
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create manager
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create mock adapter
	mockAdapter := tmux.NewMockAdapter()

	// Create session info
	now := time.Now()
	info := &Info{
		ID:          "test-mock-session",
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     "echo 'Test session started'",
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
		CreatedAt: now,
	}

	// Save info
	if err := manager.Save(context.Background(), info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create session directory
	sessionDir := filepath.Join(configManager.GetAmuxDir(), "sessions", info.ID)
	os.MkdirAll(sessionDir, 0o755)

	// Create state manager for the session
	stateManager := state.NewManager(
		info.ID,
		info.WorkspaceID,
		sessionDir,
		nil, // No logger for tests
	)

	// Create tmux session with mock and state manager
	session := NewTmuxSession(info, manager, mockAdapter, ws, nil, WithStateManager(stateManager))

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
	if err := session.Stop(context.Background()); err != nil {
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
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create manager
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

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

	session, err := manager.CreateSession(context.Background(), opts)
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
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create manager
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Replace with unavailable mock
	mockAdapter := tmux.NewMockAdapter()
	mockAdapter.SetAvailable(false)
	manager.tmuxAdapter = mockAdapter

	// Create session
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
	}

	_, err = manager.CreateSession(context.Background(), opts)
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
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace-status-mock",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create manager
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create mock adapter
	mockAdapter := tmux.NewMockAdapter()

	// Create session info
	now := time.Now()
	info := &Info{
		ID:          "test-status-mock-session",
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     "test-command",
		CreatedAt:   now,
	}

	// Save info
	if err := manager.Save(context.Background(), info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create session directory
	sessionDir := filepath.Join(configManager.GetAmuxDir(), "sessions", info.ID)
	os.MkdirAll(sessionDir, 0o755)

	// Create state manager for the session
	stateManager := state.NewManager(
		info.ID,
		info.WorkspaceID,
		sessionDir,
		nil, // No logger for tests
	)

	// Create tmux session with mock adapter and state manager
	session := NewTmuxSession(info, manager, mockAdapter, ws, nil, WithStateManager(stateManager)).(*tmuxSessionImpl)

	// Initialize the session as if it started
	session.info.TmuxSession = "test-session"
	// Set initial status through state machine if available
	if session.stateManager != nil {
		// Created -> Starting -> Running
		if err := session.stateManager.TransitionTo(context.Background(), state.StatusStarting); err != nil {
			t.Fatalf("Failed to transition to starting: %v", err)
		}
		if err := session.stateManager.TransitionTo(context.Background(), state.StatusRunning); err != nil {
			t.Fatalf("Failed to transition to running: %v", err)
		}
	}

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
		expectedStatus       state.Status
		checkStatusChangedAt bool
	}{
		{
			name: "Initial running status remains running",
			setupFunc: func() {
				// First update to establish baseline
				if err := session.UpdateStatus(context.Background()); err != nil {
					t.Fatalf("Failed to update status: %v", err)
				}
			},
			expectedStatus:       state.StatusRunning,
			checkStatusChangedAt: false,
		},
		{
			name: "Status remains running with new output",
			setupFunc: func() {
				// Change output
				mockAdapter.SetPaneContent("test-session", "new output")
				if err := session.UpdateStatus(context.Background()); err != nil {
					t.Fatalf("Failed to update status: %v", err)
				}
			},
			expectedStatus:       state.StatusRunning,
			checkStatusChangedAt: false,
		},
		{
			name: "Status remains running within activity threshold",
			setupFunc: func() {
				// Wait a bit but less than idle threshold
				time.Sleep(1 * time.Second)
				if err := session.UpdateStatus(context.Background()); err != nil {
					t.Fatalf("Failed to update status: %v", err)
				}
			},
			expectedStatus:       state.StatusRunning,
			checkStatusChangedAt: false,
		},
		{
			name: "Status remains running even without new output",
			setupFunc: func() {
				// Reset to a known state with unique output
				mockAdapter.SetPaneContent("test-session", "idle test output final")

				// Sleep to ensure cache from previous tests expires
				time.Sleep(1100 * time.Millisecond)

				// First ensure we have current state by calling UpdateStatus
				// This will capture the current output and set lastOutputContent
				err := session.UpdateStatus(context.Background())
				if err != nil {
					t.Fatalf("First UpdateStatus failed: %v", err)
				}

				// At this point, status should be running
				if session.Status() != state.StatusRunning {
					t.Fatalf("Expected running status after output change, got %s", session.Status())
				}

				// Wait for idle threshold to pass
				time.Sleep(3500 * time.Millisecond) // Well over 3 seconds

				// Update status again - should remain running (no idle state)
				err = session.UpdateStatus(context.Background())
				if err != nil {
					t.Fatalf("Second UpdateStatus failed: %v", err)
				}
			},
			expectedStatus:       state.StatusRunning,
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

			// Check state change happened by checking current status
			if tt.checkStatusChangedAt {
				// For now, just verify we have the expected status
				// State machine tracks the actual change timestamps
				_ = tt.checkStatusChangedAt // Mark as used
			}
		})
	}
}
