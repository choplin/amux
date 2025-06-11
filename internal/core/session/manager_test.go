package session

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/workspace"
)

func TestManager_CreateSession(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name:       "test-workspace",
		BaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
	}

	// Create session store
	store, err := NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}

	// Create session manager (nil ID mapper and mailbox manager for tests)
	manager := NewManager(store, wsManager, nil, nil)

	// Use mock adapter for consistent testing across platforms
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Test creating a session
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
		Command:     "claude code",
		Environment: map[string]string{
			"ANTHROPIC_API_KEY": "test-key",
		},
	}

	session, err := manager.CreateSession(opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session properties
	if session.WorkspaceID() != ws.ID {
		t.Errorf("Expected workspace ID %s, got %s", ws.ID, session.WorkspaceID())
	}

	if session.AgentID() != "claude" {
		t.Errorf("Expected agent ID 'claude', got %s", session.AgentID())
	}

	if session.Status() != StatusCreated {
		t.Errorf("Expected status %s, got %s", StatusCreated, session.Status())
	}

	// Verify session was saved to store
	loaded, err := store.Load(session.ID())
	if err != nil {
		t.Fatalf("Failed to load session from store: %v", err)
	}

	if loaded.ID != session.ID() {
		t.Errorf("Loaded session ID mismatch")
	}
}

func TestManager_GetSession(t *testing.T) {
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	ws, err := wsManager.Create(workspace.CreateOptions{
		Name: "test-workspace",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	store, err := NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	manager := NewManager(store, wsManager, nil, nil)

	// Use mock adapter for consistent testing across platforms
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create a session
	session, err := manager.CreateSession(Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get the session
	retrieved, err := manager.GetSession(session.ID())
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.ID() != session.ID() {
		t.Errorf("Retrieved session ID mismatch")
	}

	// Test getting non-existent session
	_, err = manager.GetSession("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestManager_ListSessions(t *testing.T) {
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	ws, err := wsManager.Create(workspace.CreateOptions{
		Name: "test-workspace",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	store, err := NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Ensure we start with a clean slate - list existing sessions
	existingSessions, _ := store.List()
	if len(existingSessions) > 0 {
		t.Logf("Warning: Found %d existing sessions before test", len(existingSessions))
	}

	manager := NewManager(store, wsManager, nil, nil)

	// Use mock adapter for consistent testing across platforms
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create multiple sessions
	var sessionIDs []string
	for i := 0; i < 3; i++ {
		session, err := manager.CreateSession(Options{
			WorkspaceID: ws.ID,
			AgentID:     fmt.Sprintf("agent-%d", i),
		})
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
		sessionIDs = append(sessionIDs, session.ID())

		// Verify session was saved
		savedInfo, err := store.Load(session.ID())
		if err != nil {
			t.Fatalf("Failed to load session %d after creation: %v", i, err)
		}
		if savedInfo.ID != session.ID() {
			t.Errorf("Session %d ID mismatch: expected %s, got %s", i, session.ID(), savedInfo.ID)
		}

		// Debug: List sessions after each creation
		currentSessions, _ := store.List()
		t.Logf("After creating session %d: found %d sessions in store", i, len(currentSessions))

		// Small delay to ensure file system operations complete on Windows
		time.Sleep(10 * time.Millisecond)
	}

	// List sessions
	sessions, err := manager.ListSessions()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
		// Log what we actually got
		for i, s := range sessions {
			t.Logf("Session %d: ID=%s, AgentID=%s", i, s.ID(), s.AgentID())
		}
	}

	// Verify all sessions are in the list
	found := make(map[string]bool)
	for _, session := range sessions {
		found[session.ID()] = true
	}

	for _, id := range sessionIDs {
		if !found[id] {
			t.Errorf("Session %s not found in list", id)
		}
	}
}

func TestManager_RemoveSession(t *testing.T) {
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	ws, err := wsManager.Create(workspace.CreateOptions{
		Name: "test-workspace",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	store, err := NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	manager := NewManager(store, wsManager, nil, nil)

	// Use mock adapter for consistent testing across platforms
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create a session
	session, err := manager.CreateSession(Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Can't remove running session
	ctx := context.Background()
	if err := session.Start(ctx); err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	err = manager.RemoveSession(session.ID())
	if err == nil {
		t.Error("Expected error removing running session")
	}

	// Stop session
	if err := session.Stop(); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	// Now should be able to remove
	if err := manager.RemoveSession(session.ID()); err != nil {
		t.Fatalf("Failed to remove stopped session: %v", err)
	}

	// Verify session is gone
	_, err = manager.GetSession(session.ID())
	if err == nil {
		t.Error("Expected error getting removed session")
	}
}

func TestFileStore_Operations(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Test Save and Load
	info := &Info{
		ID:          "test-session",
		WorkspaceID: "test-workspace",
		AgentID:     "claude",
		Status:      StatusCreated,
		CreatedAt:   time.Now(),
	}

	if err := store.Save(info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	loaded, err := store.Load(info.ID)
	if err != nil {
		t.Fatalf("Failed to load session info: %v", err)
	}

	if loaded.ID != info.ID {
		t.Errorf("Loaded ID mismatch: expected %s, got %s", info.ID, loaded.ID)
	}

	// Test List
	infos, err := store.List()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(infos) != 1 {
		t.Errorf("Expected 1 session, got %d", len(infos))
	}

	// Test Delete
	if err := store.Delete(info.ID); err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify deleted
	_, err = store.Load(info.ID)
	if err == nil {
		t.Error("Expected error loading deleted session")
	}
}
