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
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name:       "test-workspace",
		BaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
	}

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create session manager
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

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

	session, err := manager.CreateSession(context.Background(), opts)
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

	// Verify session was saved to manager
	loaded, err := manager.Load(context.Background(), session.ID())
	if err != nil {
		t.Fatalf("Failed to load session from manager: %v", err)
	}

	if loaded.ID != session.ID() {
		t.Errorf("Loaded session ID mismatch")
	}
}

func TestManager_CreateSessionWithNameAndDescription(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name:       "test-workspace-named",
		BaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
	}

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create session manager
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	// Use mock adapter for consistent testing across platforms
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Test creating a session with name and description
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     "echo test",
		Name:        "my-test-session",
		Description: "This is a test session",
	}

	session, err := manager.CreateSession(context.Background(), opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session properties
	info := session.Info()
	if info.Name != "my-test-session" {
		t.Errorf("Expected session name 'my-test-session', got %s", info.Name)
	}

	if info.Description != "This is a test session" {
		t.Errorf("Expected session description 'This is a test session', got %s", info.Description)
	}

	// Verify session was saved with name and description
	loaded, err := manager.Load(context.Background(), session.ID())
	if err != nil {
		t.Fatalf("Failed to load session from manager: %v", err)
	}

	if loaded.Name != "my-test-session" {
		t.Errorf("Loaded session name mismatch: expected 'my-test-session', got %s", loaded.Name)
	}

	if loaded.Description != "This is a test session" {
		t.Errorf("Loaded session description mismatch: expected 'This is a test session', got %s", loaded.Description)
	}
}

func TestManager_CreateSessionWithInitialPrompt(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name:       "test-workspace-prompt",
		BaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
	}

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create session manager
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	// Use mock adapter for consistent testing across platforms
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Test creating a session with initial prompt
	opts := Options{
		WorkspaceID:   ws.ID,
		AgentID:       "test-agent",
		Command:       "python",
		InitialPrompt: "print('Hello, World!')",
	}

	session, err := manager.CreateSession(context.Background(), opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session properties
	info := session.Info()
	if info.InitialPrompt != "print('Hello, World!')" {
		t.Errorf("Expected initial prompt 'print('Hello, World!')', got %s", info.InitialPrompt)
	}

	// Verify session was saved with initial prompt
	loaded, err := manager.Load(context.Background(), session.ID())
	if err != nil {
		t.Fatalf("Failed to load session from manager: %v", err)
	}

	if loaded.InitialPrompt != "print('Hello, World!')" {
		t.Errorf("Loaded session initial prompt mismatch: expected 'print('Hello, World!')', got %s", loaded.InitialPrompt)
	}
}

func TestManager_Get(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name:       "test-workspace-get",
		BaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
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

	// Use mock adapter
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create a session
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
		Command:     "claude code",
	}

	created, err := manager.CreateSession(context.Background(), opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test Get
	retrieved, err := manager.Get(context.Background(), ID(created.ID()))
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.ID() != created.ID() {
		t.Errorf("Retrieved session ID mismatch")
	}

	// Test Get non-existent session
	_, err = manager.Get(context.Background(), ID("non-existent"))
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestManager_ListSessions(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create test workspaces
	workspaces := make([]*workspace.Workspace, 3)
	for i := 0; i < 3; i++ {
		ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
			Name:       fmt.Sprintf("test-workspace-%d", i),
			BaseBranch: "main",
		})
		if err != nil {
			t.Fatalf("Failed to create test workspace %d: %v", i, err)
		}
		workspaces[i] = ws
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

	// Use mock adapter
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create multiple sessions
	createdSessions := make([]Session, 3)
	for i := 0; i < 3; i++ {
		opts := Options{
			WorkspaceID: workspaces[i].ID,
			AgentID:     fmt.Sprintf("agent-%d", i),
			Command:     fmt.Sprintf("echo test-%d", i),
		}

		session, err := manager.CreateSession(context.Background(), opts)
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
		createdSessions[i] = session

		// Verify the session count after each creation
		sessions, err := manager.ListSessions(context.Background())
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}
		if len(sessions) != i+1 {
			t.Errorf("After creating session %d: expected %d sessions, got %d", i, i+1, len(sessions))
		}
		t.Logf("After creating session %d: found %d sessions in manager", i, len(sessions))
	}

	// List all sessions
	sessions, err := manager.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}

	// Verify all created sessions are in the list
	for _, created := range createdSessions {
		found := false
		for _, listed := range sessions {
			if listed.ID() == created.ID() {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Created session %s not found in list", created.ID())
		}
	}
}

func TestManager_Remove(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name:       "test-workspace-remove",
		BaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
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

	// Use mock adapter
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create a session
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
		Command:     "claude code",
	}

	session, err := manager.CreateSession(context.Background(), opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	sessionID := ID(session.ID())

	// Start the session (so we can test removal of running session)
	if err := session.(TerminalSession).Start(context.Background()); err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Try to remove running session - should fail
	if err := manager.Remove(context.Background(), sessionID); err == nil {
		t.Error("Expected error removing running session")
	}

	// Stop the session
	if err := session.(TerminalSession).Stop(context.Background()); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	// Now remove should succeed
	if err := manager.Remove(context.Background(), sessionID); err != nil {
		t.Fatalf("Failed to remove stopped session: %v", err)
	}

	// Verify session is removed
	_, err = manager.Get(context.Background(), sessionID)
	if err == nil {
		t.Error("Expected error getting removed session")
	}
}

func TestManager_RemoveCompletedSession(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name:       "test-workspace-remove-completed",
		BaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
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

	// Use mock adapter
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create a session
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "test",
		Command:     "echo done",
	}

	session, err := manager.CreateSession(context.Background(), opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Simulate session completion
	info := session.Info()
	now := time.Now()
	info.StatusState.Status = StatusCompleted
	info.StatusState.StatusChangedAt = now
	info.StoppedAt = &now
	if err := manager.Save(context.Background(), info); err != nil {
		t.Fatalf("Failed to update session status: %v", err)
	}

	// Remove completed session should succeed
	if err := manager.Remove(context.Background(), ID(session.ID())); err != nil {
		t.Fatalf("Failed to remove completed session: %v", err)
	}

	// Verify session is removed
	_, err = manager.Get(context.Background(), ID(session.ID()))
	if err == nil {
		t.Error("Expected error getting removed session")
	}
}

func TestManager_CreateSessionWithoutTmux(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name:       "test-workspace-no-tmux",
		BaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
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

	// Use mock adapter that's not available
	mockAdapter := tmux.NewMockAdapter()
	mockAdapter.SetAvailable(false)
	manager.SetTmuxAdapter(mockAdapter)

	// Try to create a session - should fail
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
		Command:     "claude code",
	}

	_, err = manager.CreateSession(context.Background(), opts)
	if err == nil {
		t.Error("Expected error creating session without tmux")
	}

	// Verify the error is ErrTmuxNotAvailable
	if _, ok := err.(ErrTmuxNotAvailable); !ok {
		t.Errorf("Expected ErrTmuxNotAvailable, got %T", err)
	}
}

func TestManager_GetWithoutTmux(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name:       "test-workspace-get-no-tmux",
		BaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
	}

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create manager with available tmux first
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Use mock adapter
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create a session while tmux is available
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
		Command:     "claude code",
	}

	session, err := manager.CreateSession(context.Background(), opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Now make tmux unavailable
	mockAdapter.SetAvailable(false)

	// Clear cache to force loading from file
	manager.mu.Lock()
	delete(manager.sessions, session.ID())
	manager.mu.Unlock()

	// Try to get the session - should fail
	retrievedSession, err := manager.Get(context.Background(), ID(session.ID()))
	if err == nil {
		t.Error("Expected error getting session without tmux")
		if retrievedSession != nil {
			t.Logf("Unexpectedly got session: %v", retrievedSession.Info())
		}
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

func TestManager_StoreOperations(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

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

	if err := manager.Save(context.Background(), info); err != nil {
		t.Fatalf("Failed to save session info: %v", err)
	}

	loaded, err := manager.Load(context.Background(), info.ID)
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
	infos, err := manager.List(context.Background())
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(infos) != 1 {
		t.Errorf("Expected 1 session, got %d", len(infos))
	}

	// Test Delete
	if err := manager.Delete(context.Background(), info.ID); err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	// Verify deleted
	_, err = manager.Load(context.Background(), info.ID)
	if err == nil {
		t.Error("Expected error loading deleted session")
	}
}

func TestManager_ListSessionsWithDeletedWorkspace(t *testing.T) {
	// Setup test environment
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create a test workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name:       "test-workspace-to-delete",
		BaseBranch: "main",
	})
	if err != nil {
		t.Fatalf("Failed to create test workspace: %v", err)
	}

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create session manager
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	// Use mock adapter
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create a session in the workspace
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     "echo test",
		Name:        "session-with-deleted-workspace",
	}

	session, err := manager.CreateSession(context.Background(), opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session was created
	sessions, err := manager.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	// Delete the workspace
	if err := wsManager.Remove(context.Background(), workspace.Identifier(ws.ID)); err != nil {
		t.Fatalf("Failed to remove workspace: %v", err)
	}

	// Clear the session cache to force re-creation from stored info
	// This simulates what would happen if the manager was restarted
	manager.mu.Lock()
	manager.sessions = make(map[string]Session)
	manager.mu.Unlock()

	// List sessions again - should not fail
	sessions, err = manager.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("Failed to list sessions after workspace deletion: %v", err)
	}

	// Session should still be returned
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session after workspace deletion, got %d", len(sessions))
	}

	// Verify the session has empty workspace path
	listedSession := sessions[0]
	workspacePath := listedSession.WorkspacePath()

	// Additional debugging
	t.Logf("Session workspace path after deletion: %s", workspacePath)
	t.Logf("Session type: %T", listedSession)

	if workspacePath != "" {
		t.Errorf("Expected empty workspace path for deleted workspace, got %s", workspacePath)
	}

	// Verify session info is still accessible
	info := listedSession.Info()
	if info.ID != session.ID() {
		t.Errorf("Session ID mismatch: expected %s, got %s", session.ID(), info.ID)
	}
	if info.WorkspaceID != ws.ID {
		t.Errorf("Workspace ID should still be preserved: expected %s, got %s", ws.ID, info.WorkspaceID)
	}
	if info.Name != "session-with-deleted-workspace" {
		t.Errorf("Session name mismatch: expected 'session-with-deleted-workspace', got %s", info.Name)
	}
}
