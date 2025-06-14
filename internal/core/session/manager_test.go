package session

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/idmap"
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

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create session manager
	manager := NewManager(store, wsManager, idMapper)

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

func TestManager_CreateSessionWithNameAndDescription(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name:       "test-workspace-named",
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

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create session manager
	manager := NewManager(store, wsManager, idMapper)

	// Use mock adapter for consistent testing across platforms
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Test creating a session with name and description
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
		Command:     "claude code",
		Name:        "debug-session",
		Description: "Debugging authentication issues",
	}

	session, err := manager.CreateSession(opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session properties
	info := session.Info()
	if info.Name != "debug-session" {
		t.Errorf("Expected name 'debug-session', got %s", info.Name)
	}
	if info.Description != "Debugging authentication issues" {
		t.Errorf("Expected description 'Debugging authentication issues', got %s", info.Description)
	}

	// Verify session was saved to store with name and description
	loaded, err := store.Load(session.ID())
	if err != nil {
		t.Fatalf("Failed to load session from store: %v", err)
	}

	if loaded.Name != "debug-session" {
		t.Errorf("Loaded session name mismatch: expected 'debug-session', got %s", loaded.Name)
	}
	if loaded.Description != "Debugging authentication issues" {
		t.Errorf("Loaded session description mismatch")
	}
}

func TestManager_CreateSessionWithInitialPrompt(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name:       "test-workspace-prompt",
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

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create session manager
	manager := NewManager(store, wsManager, idMapper)

	// Use mock adapter for consistent testing
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Test creating a session with initial prompt
	testPrompt := "Please analyze the codebase and suggest improvements"
	opts := Options{
		WorkspaceID:   ws.ID,
		AgentID:       "claude",
		Command:       "claude code",
		InitialPrompt: testPrompt,
	}

	session, err := manager.CreateSession(opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session info includes initial prompt
	info := session.Info()
	if info.InitialPrompt != testPrompt {
		t.Errorf("Expected initial prompt '%s', got '%s'", testPrompt, info.InitialPrompt)
	}

	// Verify session was saved with initial prompt
	loaded, err := store.Load(session.ID())
	if err != nil {
		t.Fatalf("Failed to load session from store: %v", err)
	}

	if loaded.InitialPrompt != testPrompt {
		t.Errorf("Loaded initial prompt mismatch: expected '%s', got '%s'", testPrompt, loaded.InitialPrompt)
	}
}

func TestManager_Get(t *testing.T) {
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

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	manager := NewManager(store, wsManager, idMapper)

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
	retrieved, err := manager.Get(ID(session.ID()))
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.ID() != session.ID() {
		t.Errorf("Retrieved session ID mismatch")
	}

	// Test getting non-existent session
	_, err = manager.Get(ID("non-existent"))
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

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	manager := NewManager(store, wsManager, idMapper)

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

func TestManager_Remove(t *testing.T) {
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

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	manager := NewManager(store, wsManager, idMapper)

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

	err = manager.Remove(ID(session.ID()))
	if err == nil {
		t.Error("Expected error removing running session")
	}

	// Stop session
	if err := session.Stop(); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	// Now should be able to remove
	if err := manager.Remove(ID(session.ID())); err != nil {
		t.Fatalf("Failed to remove stopped session: %v", err)
	}

	// Verify session is gone
	_, err = manager.Get(ID(session.ID()))
	if err == nil {
		t.Error("Expected error getting removed session")
	}
}

func TestManager_CreateSessionWithoutTmux(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name:       "test-workspace-no-tmux",
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

	// Create session manager without tmux adapter
	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	manager := NewManager(store, wsManager, idMapper)
	manager.SetTmuxAdapter(nil) // Explicitly set to nil to simulate no tmux

	// Test creating a session without tmux
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
		Command:     "claude code",
	}

	_, err = manager.CreateSession(opts)
	if err == nil {
		t.Fatal("Expected error when creating session without tmux")
	}

	// Verify we get the correct error type
	if _, ok := err.(ErrTmuxNotAvailable); !ok {
		t.Errorf("Expected ErrTmuxNotAvailable, got %T: %v", err, err)
	}
}

func TestManager_GetWithoutTmux(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(workspace.CreateOptions{
		Name:       "test-workspace-get-no-tmux",
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

	// First create a session with tmux available
	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	manager := NewManager(store, wsManager, idMapper)
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	session, err := manager.CreateSession(Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
		Command:     "claude code",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	sessionID := session.ID()

	// Create a new manager without tmux to simulate fresh start
	// This tests the case where sessions are persisted but tmux is not available on restart
	manager2 := NewManager(store, wsManager, idMapper)
	manager2.SetTmuxAdapter(nil)

	// Try to get the session without tmux
	_, err = manager2.Get(ID(sessionID))
	if err == nil {
		t.Fatal("Expected error when getting session without tmux")
	}

	// Verify we get the correct error type
	if _, ok := err.(ErrTmuxNotAvailable); !ok {
		t.Errorf("Expected ErrTmuxNotAvailable, got %T: %v", err, err)
	}
}

func TestFileStore_Operations(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Test Save and Load
	now := time.Now()
	info := &Info{
		ID:          "test-session",
		WorkspaceID: "test-workspace",
		AgentID:     "claude",
		StatusState: StatusState{
			Status:          StatusCreated,
			StatusChangedAt: now,
			LastOutputTime:  now,
		},
		CreatedAt:   now,
		Name:        "test-session-name",
		Description: "Test session description",
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
	if loaded.Name != info.Name {
		t.Errorf("Loaded Name mismatch: expected %s, got %s", info.Name, loaded.Name)
	}
	if loaded.Description != info.Description {
		t.Errorf("Loaded Description mismatch: expected %s, got %s", info.Description, loaded.Description)
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
