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
	ws, err := wsManager.Create(workspace.CreateOptions{
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
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, idMapper)
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
	if err := manager.Save(info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create tmux session
	session := NewTmuxSession(info, manager, tmuxAdapter, ws)

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
	if err := session.Stop(); err != nil {
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
	ws, err := wsManager.Create(workspace.CreateOptions{
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
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, idMapper)
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
	if err := manager.Save(info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create tmux session
	session := NewTmuxSession(info, manager, tmuxAdapter, ws)

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
	if err := session.Stop(); err != nil {
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
	ws, err := wsManager.Create(workspace.CreateOptions{
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
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, idMapper)
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
	if err := manager.Save(info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create tmux session
	session := NewTmuxSession(info, manager, tmuxAdapter, ws)

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
	if err := session.Stop(); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	// After stop, status should be stopped
	if status := session.Status(); status != StatusStopped {
		t.Errorf("Expected status after stop to be stopped, got %s", status)
	}
}
