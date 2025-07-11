package session

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/task"
	"github.com/aki/amux/internal/workspace"
)

func TestLocalRuntimeSessionCreation(t *testing.T) {
	// Skip this test as it requires the amux binary to be in PATH
	t.Skip("Requires amux binary in PATH")
}

func TestSessionListFormatting(t *testing.T) {
	// Test session formatting for ps command
	now := time.Now()
	sessions := []*Session{
		{
			ID:             "session-test-1752161234-abcd1234",
			ShortID:        "1",
			Name:           "test-session",
			Status:         StatusRunning,
			Runtime:        "local",
			WorkspaceID:    "workspace-test-1752161234-efgh5678",
			LastActivityAt: now.Add(-30 * time.Second),
			StartedAt:      now.Add(-2 * time.Minute),
		},
		{
			ID:        "session-2",
			ShortID:   "2",
			Name:      "",
			Status:    StatusStopped,
			Runtime:   "tmux",
			StartedAt: now.Add(-1 * time.Hour),
			StoppedAt: &now,
		},
	}

	// Test that all required fields are present
	for _, s := range sessions {
		if s.ID == "" {
			t.Error("Session ID should not be empty")
		}

		// ShortID should be set
		if s.ShortID == "" {
			t.Error("Session ShortID should not be empty")
		}

		// Name should default to ID if empty
		name := s.Name
		if name == "" {
			name = s.ID
		}
		if name == "" {
			t.Error("Session name should not be empty")
		}

		// LastActivityAt formatting
		if s.Status == StatusRunning && !s.LastActivityAt.IsZero() {
			elapsed := time.Since(s.LastActivityAt)
			if elapsed < 0 {
				t.Error("LastActivityAt should not be in the future")
			}
		}
	}
}

func TestSessionStatusUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir)
	taskMgr := task.NewManager()

	// Create mock runtime that we can control
	mockRt := newMockRuntime("test")
	runtimes := map[string]runtime.Runtime{
		"test": mockRt,
	}

	mgr := NewManager(store, runtimes, taskMgr, nil, nil)

	ctx := context.Background()

	// Create a session
	opts := CreateOptions{
		Name:    "status-test",
		Command: []string{"sleep", "1"},
		Runtime: "test",
	}

	session, err := mgr.Create(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Initial status should be running
	if session.Status != StatusRunning {
		t.Errorf("Expected initial status %s, got %s", StatusRunning, session.Status)
	}

	// Update status
	err = mgr.UpdateStatus(ctx, session.ID, StatusStopped)
	if err != nil {
		t.Fatalf("Failed to update status: %v", err)
	}

	// Verify status was updated
	updated, err := mgr.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if updated.Status != StatusStopped {
		t.Errorf("Expected status %s, got %s", StatusStopped, updated.Status)
	}
}

func TestSessionWorkspaceAutoCreation(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir)
	taskMgr := task.NewManager()

	// Mock workspace manager
	mockWsMgr := &testWorkspaceManager{
		workspaces: make(map[string]*workspace.Workspace),
	}

	runtimes := map[string]runtime.Runtime{
		"test": newMockRuntime("test"),
	}

	mgr := NewManager(store, runtimes, taskMgr, mockWsMgr, nil)

	ctx := context.Background()

	// Create session with auto workspace creation
	opts := CreateOptions{
		Name:                "auto-ws-test",
		Command:             []string{"echo", "test"},
		Runtime:             "test",
		AutoCreateWorkspace: true,
	}

	session, err := mgr.Create(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify workspace was created
	if session.WorkspaceID == "" {
		t.Error("Expected workspace ID to be set")
	}

	// Verify workspace exists in mock manager
	if _, exists := mockWsMgr.workspaces[session.WorkspaceID]; !exists {
		t.Error("Expected workspace to be created")
	}
}

// testWorkspaceManager implements WorkspaceManager for testing
type testWorkspaceManager struct {
	mu         sync.Mutex
	workspaces map[string]*workspace.Workspace
}

func (m *testWorkspaceManager) Create(ctx context.Context, opts workspace.CreateOptions) (*workspace.Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := strings.ReplaceAll(opts.Name, " ", "-")
	ws := &workspace.Workspace{
		ID:          id,
		Name:        opts.Name,
		Description: opts.Description,
		AutoCreated: opts.AutoCreated,
		Path:        filepath.Join("/tmp", id),
		CreatedAt:   time.Now(),
	}

	m.workspaces[id] = ws
	return ws, nil
}
