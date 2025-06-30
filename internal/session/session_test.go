package session

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/task"
)

// mockRuntime implements runtime.Runtime for testing
type mockRuntime struct {
	name      string
	processes map[string]*mockProcess
}

func newMockRuntime(name string) *mockRuntime {
	return &mockRuntime{
		name:      name,
		processes: make(map[string]*mockProcess),
	}
}

func (r *mockRuntime) Type() string {
	return r.name
}

func (r *mockRuntime) Execute(ctx context.Context, spec runtime.ExecutionSpec) (runtime.Process, error) {
	if len(spec.Command) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	processID := fmt.Sprintf("mock-process-%d", len(r.processes)+1)
	process := &mockProcess{
		id:        processID,
		spec:      spec,
		state:     runtime.StateRunning,
		startTime: time.Now(),
		exitCh:    make(chan struct{}),
	}

	r.processes[processID] = process
	return process, nil
}

func (r *mockRuntime) Find(ctx context.Context, id string) (runtime.Process, error) {
	process, ok := r.processes[id]
	if !ok {
		return nil, fmt.Errorf("process not found: %s", id)
	}
	return process, nil
}

func (r *mockRuntime) List(ctx context.Context) ([]runtime.Process, error) {
	var processes []runtime.Process
	for _, p := range r.processes {
		processes = append(processes, p)
	}
	return processes, nil
}

func (r *mockRuntime) Validate() error {
	return nil
}

// mockProcess implements runtime.Process for testing
type mockProcess struct {
	id        string
	spec      runtime.ExecutionSpec
	state     runtime.ProcessState
	startTime time.Time
	exitCode  int
	exitCh    chan struct{}
}

func (p *mockProcess) ID() string {
	return p.id
}

func (p *mockProcess) StartTime() time.Time {
	return p.startTime
}

func (p *mockProcess) State() runtime.ProcessState {
	return p.state
}

func (p *mockProcess) ExitCode() (int, error) {
	if p.state == runtime.StateRunning {
		return -1, fmt.Errorf("process still running")
	}
	return p.exitCode, nil
}

func (p *mockProcess) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.exitCh:
		return nil
	}
}

func (p *mockProcess) Stop(ctx context.Context) error {
	if p.state != runtime.StateRunning {
		return fmt.Errorf("process not running")
	}
	p.state = runtime.StateStopped
	p.exitCode = 0
	close(p.exitCh)
	return nil
}

func (p *mockProcess) Kill(ctx context.Context) error {
	if p.state != runtime.StateRunning {
		return fmt.Errorf("process not running")
	}
	p.state = runtime.StateFailed
	p.exitCode = -1
	close(p.exitCh)
	return nil
}

func (p *mockProcess) Output() (stdout, stderr io.Reader) {
	return bytes.NewReader([]byte("mock output")), nil
}

// mockStore implements Store for testing
type mockStore struct {
	sessions map[string]*Session
}

func newMockStore() *mockStore {
	return &mockStore{
		sessions: make(map[string]*Session),
	}
}

func (s *mockStore) Save(ctx context.Context, session *Session) error {
	s.sessions[session.ID] = session
	return nil
}

func (s *mockStore) Load(ctx context.Context, id string) (*Session, error) {
	session, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return session, nil
}

func (s *mockStore) List(ctx context.Context, workspaceID string) ([]*Session, error) {
	var sessions []*Session
	for _, session := range s.sessions {
		if workspaceID == "" || session.WorkspaceID == workspaceID {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (s *mockStore) Remove(ctx context.Context, id string) error {
	delete(s.sessions, id)
	return nil
}

func (s *mockStore) GetLogs(ctx context.Context, id string) (LogReader, error) {
	return &simpleLogReader{reader: nil}, nil
}

// Test setup helpers
func setupTestManager(t *testing.T) (*manager, *mockRuntime, *mockStore) {
	store := newMockStore()
	mockLocal := newMockRuntime("local")
	mockTmux := newMockRuntime("tmux")

	runtimes := map[string]runtime.Runtime{
		"local": mockLocal,
		"tmux":  mockTmux,
	}

	taskMgr := task.NewManager() // No task file for tests

	mgr := NewManager(store, runtimes, taskMgr).(*manager)

	return mgr, mockLocal, store
}

// Tests
func TestManager_Create(t *testing.T) {
	mgr, mockRuntime, store := setupTestManager(t)
	ctx := context.Background()

	opts := CreateOptions{
		WorkspaceID: "test-workspace",
		Command:     []string{"echo", "hello"},
		Runtime:     "local",
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	// Create session
	session, err := mgr.Create(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session properties
	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}
	if session.WorkspaceID != opts.WorkspaceID {
		t.Errorf("Expected workspace ID %s, got %s", opts.WorkspaceID, session.WorkspaceID)
	}
	if session.Runtime != opts.Runtime {
		t.Errorf("Expected runtime %s, got %s", opts.Runtime, session.Runtime)
	}
	if session.Status != StatusRunning {
		t.Errorf("Expected status %s, got %s", StatusRunning, session.Status)
	}

	// Verify process was created
	if len(mockRuntime.processes) != 1 {
		t.Errorf("Expected 1 process, got %d", len(mockRuntime.processes))
	}

	// Verify session was saved
	saved, err := store.Load(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to load saved session: %v", err)
	}
	if saved.ID != session.ID {
		t.Error("Saved session ID mismatch")
	}
}

func TestManager_CreateWithTask(t *testing.T) {
	// Create temp directory for task file
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "tasks.yaml")

	// Write test task file
	taskContent := `tasks:
  test-task:
    command: "echo 'Hello from task'"
    env:
      TASK_VAR: "task_value"
    working_dir: "/tmp"
`
	if err := os.WriteFile(taskFile, []byte(taskContent), 0o644); err != nil {
		t.Fatalf("Failed to write task file: %v", err)
	}

	// Create manager with task manager
	store := newMockStore()
	mockLocal := newMockRuntime("local")
	runtimes := map[string]runtime.Runtime{
		"local": mockLocal,
	}

	taskMgr := task.NewManager()
	// Load tasks from the task definition
	tasks := []*task.Task{
		{
			Name:       "test-task",
			Command:    "echo 'Hello from task'",
			Env:        map[string]string{"TASK_VAR": "task_value"},
			WorkingDir: "/tmp",
		},
	}
	if err := taskMgr.LoadTasks(tasks); err != nil {
		t.Fatalf("Failed to load tasks: %v", err)
	}

	mgr := NewManager(store, runtimes, taskMgr).(*manager)
	ctx := context.Background()

	opts := CreateOptions{
		WorkspaceID: "test-workspace",
		TaskName:    "test-task",
		Runtime:     "local",
	}

	// Create session
	session, err := mgr.Create(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to create session with task: %v", err)
	}

	// Verify task was used
	if session.TaskName != "test-task" {
		t.Errorf("Expected task name 'test-task', got %s", session.TaskName)
	}

	// Verify environment from task
	if session.Environment["TASK_VAR"] != "task_value" {
		t.Error("Task environment not applied")
	}
}

func TestManager_Get(t *testing.T) {
	mgr, _, _ := setupTestManager(t)
	ctx := context.Background()

	// Create a session
	session, err := mgr.Create(ctx, CreateOptions{
		WorkspaceID: "test-workspace",
		Command:     []string{"echo", "test"},
		Runtime:     "local",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get the session
	retrieved, err := mgr.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("Retrieved session ID mismatch: expected %s, got %s", session.ID, retrieved.ID)
	}

	// Try to get non-existent session
	_, err = mgr.Get(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestManager_List(t *testing.T) {
	mgr, _, _ := setupTestManager(t)
	ctx := context.Background()

	// Create sessions in different workspaces
	session1, err := mgr.Create(ctx, CreateOptions{
		WorkspaceID: "workspace-1",
		Command:     []string{"echo", "1"},
		Runtime:     "local",
	})
	if err != nil {
		t.Fatalf("Failed to create session 1: %v", err)
	}

	session2, err := mgr.Create(ctx, CreateOptions{
		WorkspaceID: "workspace-2",
		Command:     []string{"echo", "2"},
		Runtime:     "local",
	})
	if err != nil {
		t.Fatalf("Failed to create session 2: %v", err)
	}

	// List all sessions
	sessions, err := mgr.List(ctx, "")
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	// List sessions for specific workspace
	sessions, err = mgr.List(ctx, "workspace-1")
	if err != nil {
		t.Fatalf("Failed to list workspace sessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session for workspace-1, got %d", len(sessions))
	}
	if sessions[0].ID != session1.ID {
		t.Error("Wrong session returned for workspace-1")
	}

	// Verify session2 is not included
	for _, s := range sessions {
		if s.ID == session2.ID {
			t.Error("Session from workspace-2 should not be included")
		}
	}
}

func TestManager_StopKill(t *testing.T) {
	mgr, mockRuntime, _ := setupTestManager(t)
	ctx := context.Background()

	// Create a session
	session, err := mgr.Create(ctx, CreateOptions{
		WorkspaceID: "test-workspace",
		Command:     []string{"sleep", "60"},
		Runtime:     "local",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Get the mock process
	process := mockRuntime.processes[session.ProcessID]

	// Test Stop
	err = mgr.Stop(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	if process.state != runtime.StateStopped {
		t.Errorf("Expected process state to be stopped, got %v", process.state)
	}

	// Create another session for kill test
	session2, err := mgr.Create(ctx, CreateOptions{
		WorkspaceID: "test-workspace",
		Command:     []string{"sleep", "60"},
		Runtime:     "local",
	})
	if err != nil {
		t.Fatalf("Failed to create session 2: %v", err)
	}

	process2 := mockRuntime.processes[session2.ProcessID]

	// Test Kill
	err = mgr.Kill(ctx, session2.ID)
	if err != nil {
		t.Fatalf("Failed to kill session: %v", err)
	}

	if process2.state != runtime.StateFailed {
		t.Errorf("Expected process state to be failed, got %v", process2.state)
	}
}

func TestManager_Remove(t *testing.T) {
	mgr, mockRuntime, store := setupTestManager(t)
	ctx := context.Background()

	// Create a session
	session, err := mgr.Create(ctx, CreateOptions{
		WorkspaceID: "test-workspace",
		Command:     []string{"echo", "test"},
		Runtime:     "local",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Stop the session first
	process := mockRuntime.processes[session.ProcessID]
	process.Stop(ctx)

	// Wait a bit for monitor goroutine to update status
	time.Sleep(100 * time.Millisecond)

	// Remove the session
	err = mgr.Remove(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to remove session: %v", err)
	}

	// Verify session is removed from store
	_, err = store.Load(ctx, session.ID)
	if err == nil {
		t.Error("Session should be removed from store")
	}

	// Verify it's also removed from in-memory map
	_, err = mgr.Get(ctx, session.ID)
	if err == nil {
		t.Error("Session should be removed from manager")
	}
}

func TestManager_Attach(t *testing.T) {
	mgr, _, _ := setupTestManager(t)
	ctx := context.Background()

	// Create a tmux session
	session, err := mgr.Create(ctx, CreateOptions{
		WorkspaceID: "test-workspace",
		Command:     []string{"bash"},
		Runtime:     "tmux",
	})
	if err != nil {
		t.Fatalf("Failed to create tmux session: %v", err)
	}

	// Note: Actual attach would fail in test environment
	// because it tries to run real tmux command
	// We're just testing the logic flow

	// Test attach to non-running session
	mgr.sessions[session.ID].Status = StatusStopped
	err = mgr.Attach(ctx, session.ID)
	if err == nil {
		t.Error("Should not be able to attach to stopped session")
	}

	// Test attach to local runtime session
	localSession, _ := mgr.Create(ctx, CreateOptions{
		WorkspaceID: "test-workspace",
		Command:     []string{"bash"},
		Runtime:     "local",
	})

	err = mgr.Attach(ctx, localSession.ID)
	if err == nil || !contains(err.Error(), "not supported") {
		t.Error("Local runtime should not support attach")
	}
}

func TestManager_IDGeneration(t *testing.T) {
	mgr, _, _ := setupTestManager(t)
	ctx := context.Background()

	// Create multiple sessions
	var sessions []*Session
	for i := 0; i < 5; i++ {
		session, err := mgr.Create(ctx, CreateOptions{
			WorkspaceID: "test-workspace",
			Command:     []string{"echo", fmt.Sprintf("test-%d", i)},
			Runtime:     "local",
		})
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
		sessions = append(sessions, session)
	}

	// Verify IDs are sequential
	for i, session := range sessions {
		expectedID := fmt.Sprintf("session-%d", i+1)
		if session.ID != expectedID {
			t.Errorf("Expected session ID %s, got %s", expectedID, session.ID)
		}
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
