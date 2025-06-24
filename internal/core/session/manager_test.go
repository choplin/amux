package session

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/session/state"
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
		AgentID:     "claude",
		Command:     "claude code",
		Name:        "debug-session",
		Description: "Debugging authentication issues",
	}

	session, err := manager.CreateSession(context.Background(), opts)
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

	// Verify session was saved to manager with name and description
	loaded, err := manager.Load(context.Background(), session.ID())
	if err != nil {
		t.Fatalf("Failed to load session from manager: %v", err)
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

	session, err := manager.CreateSession(context.Background(), opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session info includes initial prompt
	info := session.Info()
	if info.InitialPrompt != testPrompt {
		t.Errorf("Expected initial prompt '%s', got '%s'", testPrompt, info.InitialPrompt)
	}

	// Verify session was saved with initial prompt
	loaded, err := manager.Load(context.Background(), session.ID())
	if err != nil {
		t.Fatalf("Failed to load session from manager: %v", err)
	}

	if loaded.InitialPrompt != testPrompt {
		t.Errorf("Loaded initial prompt mismatch: expected '%s', got '%s'", testPrompt, loaded.InitialPrompt)
	}
}

func TestManager_Get(t *testing.T) {
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

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

	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Use mock adapter for consistent testing across platforms
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create a session
	session, err := manager.CreateSession(context.Background(), Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get the session
	retrieved, err := manager.Get(context.Background(), ID(session.ID()))
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.ID() != session.ID() {
		t.Errorf("Retrieved session ID mismatch")
	}

	// Test getting non-existent session
	_, err = manager.Get(context.Background(), ID("non-existent"))
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestManager_ListSessions(t *testing.T) {
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

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

	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Ensure we start with a clean slate - list existing sessions
	existingSessions, _ := manager.List(context.Background())
	if len(existingSessions) > 0 {
		t.Logf("Warning: Found %d existing sessions before test", len(existingSessions))
	}

	// Use mock adapter for consistent testing across platforms
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create multiple sessions
	var sessionIDs []string
	for i := 0; i < 3; i++ {
		session, err := manager.CreateSession(context.Background(), Options{
			WorkspaceID: ws.ID,
			AgentID:     fmt.Sprintf("agent-%d", i),
		})
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
		sessionIDs = append(sessionIDs, session.ID())

		// Verify session was saved
		savedInfo, err := manager.Load(context.Background(), session.ID())
		if err != nil {
			t.Fatalf("Failed to load session %d after creation: %v", i, err)
		}
		if savedInfo.ID != session.ID() {
			t.Errorf("Session %d ID mismatch: expected %s, got %s", i, session.ID(), savedInfo.ID)
		}

		// Debug: List sessions after each creation
		currentSessions, _ := manager.List(context.Background())
		t.Logf("After creating session %d: found %d sessions in manager", i, len(currentSessions))

		// Small delay to ensure file system operations complete on Windows
		time.Sleep(10 * time.Millisecond)
	}

	// List sessions
	sessions, err := manager.ListSessions(context.Background())
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

	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Use mock adapter for consistent testing across platforms
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create a session
	session, err := manager.CreateSession(context.Background(), Options{
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

	err = manager.Remove(context.Background(), ID(session.ID()))
	if err == nil {
		t.Error("Expected error removing running session")
	}

	// Stop session
	if err := session.Stop(context.Background()); err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	// Now should be able to remove
	if err := manager.Remove(context.Background(), ID(session.ID())); err != nil {
		t.Fatalf("Failed to remove stopped session: %v", err)
	}

	// Verify session is gone
	_, err = manager.Get(context.Background(), ID(session.ID()))
	if err == nil {
		t.Error("Expected error getting removed session")
	}

	// Verify tmux session was killed
	tmuxSession := session.Info().TmuxSession
	if mockAdapter.SessionExists(tmuxSession) {
		t.Error("Expected tmux session to be killed after removal")
	}
}

func TestManager_RemoveCompletedSession(t *testing.T) {
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace-completed",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Use mock adapter
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create and start a session
	session, err := manager.CreateSession(context.Background(), Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	ctx := context.Background()
	if err := session.Start(ctx); err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Manually set status to completed (simulating command completion)
	// We need to update the session's internal state, not just the store
	// Cast to internal type to access internal methods
	tmuxSess := session.(*tmuxSessionImpl)

	// Get tmux session name before updating status
	tmuxSess.mu.Lock()
	tmuxSessionName := tmuxSess.info.TmuxSession
	tmuxSess.mu.Unlock()

	// Transition to completed state through StateManager if available
	if tmuxSess.stateManager != nil {
		if err := tmuxSess.stateManager.TransitionTo(context.Background(), state.StatusCompleted); err != nil {
			t.Fatalf("Failed to transition to completed state: %v", err)
		}
	}

	// Also update internal state
	tmuxSess.mu.Lock()
	tmuxSess.info.StatusState.Status = StatusCompleted
	tmuxSess.info.StatusState.StatusChangedAt = time.Now()
	tmuxSess.mu.Unlock()

	// Save to manager
	if err := manager.Save(context.Background(), tmuxSess.info); err != nil {
		t.Fatalf("Failed to save completed status: %v", err)
	}

	// Ensure tmux session exists before removal
	if !mockAdapter.SessionExists(tmuxSessionName) {
		t.Fatal("Expected tmux session to exist before removal")
	}

	// Remove completed session
	if err := manager.Remove(context.Background(), ID(session.ID())); err != nil {
		t.Fatalf("Failed to remove completed session: %v", err)
	}

	// Verify session is gone
	_, err = manager.Get(context.Background(), ID(session.ID()))
	if err == nil {
		t.Error("Expected error getting removed session")
	}

	// Verify tmux session was killed
	if mockAdapter.SessionExists(tmuxSessionName) {
		t.Error("Expected tmux session to be killed after removing completed session")
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

	// Create session manager without tmux adapter
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	manager.SetTmuxAdapter(nil) // Explicitly set to nil to simulate no tmux

	// Test creating a session without tmux
	opts := Options{
		WorkspaceID: ws.ID,
		AgentID:     "claude",
		Command:     "claude code",
	}

	_, err = manager.CreateSession(context.Background(), opts)
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

	// First create a session with tmux available
	manager, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	session, err := manager.CreateSession(context.Background(), Options{
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
	manager2, err := NewManager(configManager.GetAmuxDir(), wsManager, nil, idMapper)
	if err != nil {
		t.Fatalf("Failed to create second manager: %v", err)
	}
	manager2.SetTmuxAdapter(nil)

	// Try to get the session without tmux
	_, err = manager2.Get(context.Background(), ID(sessionID))
	if err == nil {
		t.Fatal("Expected error when getting session without tmux")
	}

	// Verify we get the correct error type
	if _, ok := err.(ErrTmuxNotAvailable); !ok {
		t.Errorf("Expected ErrTmuxNotAvailable, got %T: %v", err, err)
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
	// Setup
	_, wsManager, configManager := setupTestEnvironment(t)

	// Create workspace
	ws, err := wsManager.Create(context.Background(), workspace.CreateOptions{
		Name: "test-workspace-to-delete",
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

	// Use mock adapter
	mockAdapter := tmux.NewMockAdapter()
	manager.SetTmuxAdapter(mockAdapter)

	// Create session
	session, err := manager.CreateSession(context.Background(), Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
		Command:     "test-command",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	sessionID := session.ID()

	// Verify session is created
	if session.Status() != StatusCreated {
		t.Errorf("Expected created status, got %s", session.Status())
	}

	// Delete the workspace
	if err := wsManager.Remove(context.Background(), workspace.Identifier(ws.ID)); err != nil {
		t.Fatalf("Failed to remove workspace: %v", err)
	}

	// Clear cache to force re-loading
	manager.mu.Lock()
	delete(manager.sessions, sessionID)
	manager.mu.Unlock()

	// List sessions - should not fail
	sessions, err := manager.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	// Should have one session
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	// Session should be orphaned
	orphanedSession := sessions[0]
	info := orphanedSession.Info()
	if info.StatusState.Status != StatusOrphaned {
		t.Errorf("Expected orphaned status, got %s", info.StatusState.Status)
	}

	// Error should indicate workspace not found
	if !strings.Contains(info.Error, "workspace not found") {
		t.Errorf("Expected error to contain 'workspace not found', got: %s", info.Error)
	}

	// Workspace path should be empty
	if orphanedSession.WorkspacePath() != "" {
		t.Errorf("Expected empty workspace path for orphaned session, got: %s", orphanedSession.WorkspacePath())
	}
}
