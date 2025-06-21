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

func TestTmuxSession_StartStop(t *testing.T) {
	// Skip if tmux not available
	tmuxAdapter, err := tmux.NewAdapter()
	if err != nil || !tmuxAdapter.IsAvailable() {
		t.Skip("tmux not available")
	}

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

	// Create session info
	now := time.Now()
	info := &Info{
		ID:          "test-tmux-session",
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		StatusState: StatusState{
			Status:          StatusCreated,
			StatusChangedAt: now,
			LastOutputTime:  now,
		},
		Command:   "echo 'Test session started'",
		CreatedAt: now,
	}

	// Save info
	if err := manager.Save(context.Background(), info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create tmux session
	session := NewTmuxSession(info, manager, tmuxAdapter, ws, nil, nil)

	// Start session
	ctx := context.Background()
	if err := session.Start(ctx); err != nil {
		t.Fatalf("Failed to start tmux session: %v", err)
	}

	// Verify running
	if !session.Status().IsRunning() {
		t.Errorf("Expected running status, got %s", session.Status())
	}

	// Wait a bit for session to establish
	time.Sleep(100 * time.Millisecond)

	// Get output
	output, err := session.GetOutput(0)
	if err != nil {
		t.Errorf("Failed to get output: %v", err)
	} else {
		t.Logf("Session output: %s", string(output))
	}

	// Stop session
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	// Verify stopped
	if session.Status() != StatusStopped {
		t.Errorf("Expected status %s, got %s", StatusStopped, session.Status())
	}
}

func TestTmuxSession_WithInitialPrompt(t *testing.T) {
	// Skip if tmux not available
	tmuxAdapter, err := tmux.NewAdapter()
	if err != nil || !tmuxAdapter.IsAvailable() {
		t.Skip("tmux not available")
	}

	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace-prompt",
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

	// Create session info with initial prompt
	testPrompt := "echo 'Initial prompt executed'"
	now := time.Now()
	info := &Info{
		ID:          "test-tmux-prompt-session",
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		StatusState: StatusState{
			Status:          StatusCreated,
			StatusChangedAt: now,
			LastOutputTime:  now,
		},
		Command:       "bash", // Start bash to receive the prompt
		InitialPrompt: testPrompt,
		CreatedAt:     now,
	}

	// Save info
	if err := manager.Save(context.Background(), info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create tmux session
	session := NewTmuxSession(info, manager, tmuxAdapter, ws, nil, nil)

	// Start session
	ctx := context.Background()
	if err := session.Start(ctx); err != nil {
		t.Fatalf("Failed to start tmux session: %v", err)
	}

	// Verify running
	if !session.Status().IsRunning() {
		t.Errorf("Expected running status, got %s", session.Status())
	}

	// Wait for initial prompt to be sent and processed
	time.Sleep(500 * time.Millisecond)

	// Get output
	output, err := session.GetOutput(0)
	if err != nil {
		t.Errorf("Failed to get output: %v", err)
	} else {
		outputStr := string(output)
		t.Logf("Session output:\n%s", outputStr)

		// Verify initial prompt was executed
		if !strings.Contains(outputStr, "Initial prompt executed") {
			t.Errorf("Initial prompt not found in output")
		}
	}

	// Stop session
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}
}

func TestTmuxSession_StatusTracking(t *testing.T) {
	// Skip if tmux not available
	tmuxAdapter, err := tmux.NewAdapter()
	if err != nil || !tmuxAdapter.IsAvailable() {
		t.Skip("tmux not available")
	}

	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace-status",
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

	// Create session info
	now := time.Now()
	info := &Info{
		ID:          "test-status-session",
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		StatusState: StatusState{
			Status:          StatusCreated,
			StatusChangedAt: now,
			LastOutputTime:  now,
		},
		Command:   "bash",
		CreatedAt: now,
	}

	// Save info
	if err := manager.Save(context.Background(), info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create tmux session
	session := NewTmuxSession(info, manager, tmuxAdapter, ws, nil, nil)

	// Initial status should be created
	if status := session.Status(); status != StatusCreated {
		t.Errorf("Expected initial status to be created, got %s", status)
	}

	// Start session
	ctx := context.Background()
	if err := session.Start(ctx); err != nil {
		t.Fatalf("Failed to start tmux session: %v", err)
	}

	// After start, status should be working
	if status := session.Status(); status != StatusWorking {
		t.Errorf("Expected status after start to be working, got %s", status)
	}

	// Get output - this should keep status as working
	_, err = session.GetOutput(0)
	if err != nil {
		t.Errorf("Failed to get output: %v", err)
	}

	// Status should still be working
	if status := session.Status(); status != StatusWorking {
		t.Errorf("Expected status after output to be working, got %s", status)
	}

	// Stop session
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	// After stop, status should be stopped
	if status := session.Status(); status != StatusStopped {
		t.Errorf("Expected status after stop to be stopped, got %s", status)
	}
}

func TestTmuxSession_WithEnvironment(t *testing.T) {
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace-env",
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

	// Use mock adapter for predictable testing
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create session info with environment variables
	now := time.Now()
	info := &Info{
		ID:          "test-env-session",
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		StatusState: StatusState{
			Status:          StatusCreated,
			StatusChangedAt: now,
			LastOutputTime:  now,
		},
		Command: "/bin/bash", // Use bash as command
		Environment: map[string]string{
			"CUSTOM_VAR":  "custom_value",
			"ANOTHER_VAR": "another_value",
		},
		CreatedAt: now,
	}

	// Save info
	if err := manager.Save(context.Background(), info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create tmux session
	session := NewTmuxSession(info, manager, mockAdapter, ws, nil, nil)

	// Start session
	ctx := context.Background()
	if err := session.Start(ctx); err != nil {
		t.Fatalf("Failed to start tmux session: %v", err)
	}

	// Get the session info to check tmux session name
	sessionInfo := session.Info()
	if sessionInfo.TmuxSession == "" {
		t.Fatal("No tmux session name set")
	}

	// Check environment variables were set in mock adapter
	env := mockAdapter.GetSessionEnvironment(sessionInfo.TmuxSession)
	if env == nil {
		t.Fatal("No environment found for session")
	}

	// Test AMUX standard environment variables
	amuxVars := []struct {
		name     string
		expected string
	}{
		{"AMUX_WORKSPACE_ID", ws.ID},
		{"AMUX_WORKSPACE_PATH", ws.Path},
		{"AMUX_SESSION_ID", info.ID},
		{"AMUX_AGENT_ID", info.AgentID},
	}

	for _, v := range amuxVars {
		if env[v.name] != v.expected {
			t.Errorf("Environment variable %s not set correctly. Expected %s, got %s", v.name, v.expected, env[v.name])
		}
	}

	// Test custom environment variables
	for key, expectedValue := range info.Environment {
		if env[key] != expectedValue {
			t.Errorf("Custom environment variable %s not set correctly. Expected %s, got %s", key, expectedValue, env[key])
		}
	}

	// Stop session
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}
}

func TestTmuxSession_WithShellAndWindowName(t *testing.T) {
	// Skip if tmux not available
	tmuxAdapter, err := tmux.NewAdapter()
	if err != nil || !tmuxAdapter.IsAvailable() {
		t.Skip("tmux not available")
	}

	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace-shell",
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

	// Create session info with custom shell and window name
	now := time.Now()
	info := &Info{
		ID:          "test-shell-window-session",
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		StatusState: StatusState{
			Status:          StatusCreated,
			StatusChangedAt: now,
			LastOutputTime:  now,
		},
		Command:   "echo 'Custom shell started'",
		CreatedAt: now,
	}

	// Save info
	if err := manager.Save(context.Background(), info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create tmux session
	session := NewTmuxSession(info, manager, tmuxAdapter, ws, nil, nil)

	// Start session
	ctx := context.Background()
	if err := session.Start(ctx); err != nil {
		t.Fatalf("Failed to start tmux session: %v", err)
	}

	// Wait for session to be ready
	time.Sleep(200 * time.Millisecond)

	// Get output to verify command was executed
	output, err := session.GetOutput(0)
	if err != nil {
		t.Errorf("Failed to get output: %v", err)
	} else {
		outputStr := string(output)
		t.Logf("Session output with custom shell:\n%s", outputStr)

		// Verify the command was executed
		if !strings.Contains(outputStr, "Custom shell started") {
			t.Errorf("Command not executed in custom shell. Output: %s", outputStr)
		}
	}

	// Stop session
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}
}
