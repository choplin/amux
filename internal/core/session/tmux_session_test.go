package session

import (
	"context"
	"testing"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
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

	// Create store
	store, err := NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Create session info
	info := &Info{
		ID:          "test-tmux-session",
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Status:      StatusCreated,
		Command:     "echo 'Test session started'",
		CreatedAt:   time.Now(),
	}

	// Save info
	if err := store.Save(info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	// Create tmux session
	session := NewTmuxSession(info, store, tmuxAdapter, ws)

	// Start session
	ctx := context.Background()
	if err := session.Start(ctx); err != nil {
		t.Fatalf("Failed to start tmux session: %v", err)
	}

	// Verify running
	if session.Status() != StatusRunning {
		t.Errorf("Expected status %s, got %s", StatusRunning, session.Status())
	}

	// Wait a bit for session to establish
	time.Sleep(100 * time.Millisecond)

	// Get output
	output, err := session.GetOutput()
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
