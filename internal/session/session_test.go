package session

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/task"
	"github.com/aki/amux/internal/workspace"
)

// mockRuntime implements runtime.Runtime for testing
type mockRuntime struct {
	mu        sync.RWMutex
	name      string
	processes map[string]runtime.Process
}

func newMockRuntime(name string) *mockRuntime {
	return &mockRuntime{
		name:      name,
		processes: make(map[string]runtime.Process),
	}
}

func (r *mockRuntime) Type() string {
	return r.name
}

func (r *mockRuntime) Execute(ctx context.Context, spec runtime.ExecutionSpec) (runtime.Process, error) {
	if len(spec.Command) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	processID := fmt.Sprintf("mock-process-%d", len(r.processes)+1)
	now := time.Now()
	process := &mockProcess{
		id:             processID,
		spec:           spec,
		state:          runtime.StateRunning,
		startTime:      now,
		exitCh:         make(chan struct{}),
		lastActivityAt: now,
	}

	r.processes[processID] = process

	// Important: Store session ID to process mapping for Stop/Kill
	// In real implementation, this would be done by the runtime
	return process, nil
}

func (r *mockRuntime) Find(ctx context.Context, id string) (runtime.Process, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	process, ok := r.processes[id]
	if !ok {
		return nil, fmt.Errorf("process not found: %s", id)
	}
	return process, nil
}

func (r *mockRuntime) List(ctx context.Context) ([]runtime.Process, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var processes []runtime.Process
	for _, p := range r.processes {
		processes = append(processes, p)
	}
	return processes, nil
}

func (r *mockRuntime) Validate() error {
	return nil
}

// Stop implements StoppableRuntime
func (r *mockRuntime) Stop(ctx context.Context, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find process by session ID
	for _, p := range r.processes {
		if mp, ok := p.(*mockProcess); ok && mp.spec.SessionID == sessionID {
			return mp.Stop(ctx)
		}
	}
	return fmt.Errorf("session not found: %s", sessionID)
}

// Kill implements KillableRuntime
func (r *mockRuntime) Kill(ctx context.Context, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find process by session ID
	for _, p := range r.processes {
		if mp, ok := p.(*mockProcess); ok && mp.spec.SessionID == sessionID {
			return mp.Kill(ctx)
		}
	}
	return fmt.Errorf("session not found: %s", sessionID)
}

// SendInput implements InputSendingRuntime
func (r *mockRuntime) SendInput(ctx context.Context, sessionID string, input string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find process by session ID
	for _, p := range r.processes {
		if mp, ok := p.(*mockProcess); ok && mp.spec.SessionID == sessionID {
			// For basic mockProcess, we don't support input
			return fmt.Errorf("input not supported")
		}
	}
	return fmt.Errorf("session not found: %s", sessionID)
}

// mockProcess implements runtime.Process for testing
type mockProcess struct {
	mu             sync.RWMutex
	id             string
	spec           runtime.ExecutionSpec
	state          runtime.ProcessState
	startTime      time.Time
	exitCode       int
	exitCh         chan struct{}
	lastActivityAt time.Time
}

func (p *mockProcess) ID() string {
	return p.id
}

func (p *mockProcess) StartTime() time.Time {
	return p.startTime
}

func (p *mockProcess) State() runtime.ProcessState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

func (p *mockProcess) ExitCode() (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
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
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state != runtime.StateRunning {
		return fmt.Errorf("process not running")
	}
	p.state = runtime.StateStopped
	p.exitCode = 0
	select {
	case <-p.exitCh:
		// Already closed
	default:
		close(p.exitCh)
	}
	return nil
}

func (p *mockProcess) Kill(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state != runtime.StateRunning {
		return fmt.Errorf("process not running")
	}
	p.state = runtime.StateFailed
	p.exitCode = -1
	select {
	case <-p.exitCh:
		// Already closed
	default:
		close(p.exitCh)
	}
	return nil
}

func (p *mockProcess) Output() (stdout, stderr io.Reader) {
	return bytes.NewReader([]byte("mock output")), nil
}

func (p *mockProcess) Metadata() runtime.Metadata {
	return nil
}

// GetLastActivityAt implements runtime.ActivityMonitor for testing
func (p *mockProcess) GetLastActivityAt() (time.Time, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.lastActivityAt.IsZero() {
		return p.startTime, nil
	}
	return p.lastActivityAt, nil
}

// mockInputSenderProcess implements both runtime.Process and runtime.InputSender for testing
type mockInputSenderProcess struct {
	*mockProcess
	mu        sync.RWMutex
	lastInput string
}

func (p *mockInputSenderProcess) SendInput(input string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lastInput = input
	return nil
}

// mockRuntimeWithInputSender is a mock runtime that creates InputSender processes
type mockRuntimeWithInputSender struct {
	*mockRuntime
}

func (r *mockRuntimeWithInputSender) Execute(ctx context.Context, spec runtime.ExecutionSpec) (runtime.Process, error) {
	if len(spec.Command) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	processID := fmt.Sprintf("mock-process-%d", len(r.processes)+1)
	process := &mockInputSenderProcess{
		mockProcess: &mockProcess{
			id:        processID,
			spec:      spec,
			state:     runtime.StateRunning,
			startTime: time.Now(),
			exitCh:    make(chan struct{}),
		},
	}

	r.processes[processID] = process
	return process, nil
}

// SendInput overrides mockRuntime's SendInput to support input
func (r *mockRuntimeWithInputSender) SendInput(ctx context.Context, sessionID string, input string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find process by session ID
	for _, p := range r.processes {
		if mp, ok := p.(*mockInputSenderProcess); ok && mp.spec.SessionID == sessionID {
			return mp.SendInput(input)
		}
	}
	return fmt.Errorf("session not found: %s", sessionID)
}

// mockStore implements Store for testing
type mockStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func newMockStore() *mockStore {
	return &mockStore{
		sessions: make(map[string]*Session),
	}
}

func (s *mockStore) Save(ctx context.Context, session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
	return nil
}

func (s *mockStore) Load(ctx context.Context, id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	return session, nil
}

func (s *mockStore) List(ctx context.Context, workspaceID string) ([]*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sessions []*Session
	for _, session := range s.sessions {
		if workspaceID == "" || session.WorkspaceID == workspaceID {
			sessions = append(sessions, session)
		}
	}
	return sessions, nil
}

func (s *mockStore) Remove(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

func (s *mockStore) GetLogs(ctx context.Context, id string) (LogReader, error) {
	return &simpleLogReader{reader: nil}, nil
}

// mockWorkspaceManager implements WorkspaceManager interface for testing
type mockWorkspaceManager struct {
	mu         sync.RWMutex
	workspaces map[string]*workspace.Workspace
	idCounter  int
}

func newMockWorkspaceManager() *mockWorkspaceManager {
	return &mockWorkspaceManager{
		workspaces: make(map[string]*workspace.Workspace),
	}
}

func (m *mockWorkspaceManager) Create(ctx context.Context, opts workspace.CreateOptions) (*workspace.Workspace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.idCounter++
	ws := &workspace.Workspace{
		ID:          fmt.Sprintf("ws-%d", m.idCounter),
		Name:        opts.Name,
		Description: opts.Description,
		AutoCreated: opts.AutoCreated,
		CreatedAt:   time.Now(),
	}
	m.workspaces[ws.ID] = ws
	return ws, nil
}

func (m *mockWorkspaceManager) Get(ctx context.Context, id workspace.ID) (*workspace.Workspace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ws, ok := m.workspaces[string(id)]
	if !ok {
		return nil, fmt.Errorf("workspace not found")
	}
	return ws, nil
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
	// For testing, we'll use nil workspace manager and check for nil in the code
	mgr := NewManager(store, runtimes, taskMgr, nil, nil).(*manager)

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

	// For testing, we'll use nil workspace manager
	mgr := NewManager(store, runtimes, taskMgr, nil, nil).(*manager)
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
	mgr, _, _ := setupTestManager(t)
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

	// Test Stop
	err = mgr.Stop(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	// Check session status after stop
	session, err = mgr.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get session after stop: %v", err)
	}
	if session.Status != StatusStopped {
		t.Errorf("Expected session status to be stopped, got %v", session.Status)
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

	// Test Kill
	err = mgr.Kill(ctx, session2.ID)
	if err != nil {
		t.Fatalf("Failed to kill session: %v", err)
	}

	// Check session status after kill
	session2, err = mgr.Get(ctx, session2.ID)
	if err != nil {
		t.Fatalf("Failed to get session2 after kill: %v", err)
	}
	if session2.Status != StatusFailed {
		t.Errorf("Expected session status to be failed, got %v", session2.Status)
	}
}

func TestManager_Remove(t *testing.T) {
	mgr, _, store := setupTestManager(t)
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
	err = mgr.Stop(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to stop session: %v", err)
	}

	// Wait a bit for status update
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

// TestManager_AutoCreateWorkspace tests the auto-workspace creation feature
func TestManager_AutoCreateWorkspace(t *testing.T) {
	store := newMockStore()
	mockLocal := newMockRuntime("local")

	runtimes := map[string]runtime.Runtime{
		"local": mockLocal,
	}

	taskMgr := task.NewManager()
	// Use a mock workspace manager for this test
	wsMgr := newMockWorkspaceManager()
	mgr := NewManager(store, runtimes, taskMgr, wsMgr, nil).(*manager)
	ctx := context.Background()

	// Test auto-create workspace
	opts := CreateOptions{
		AutoCreateWorkspace: true,
		Command:             []string{"echo", "test"},
		Runtime:             "local",
	}

	sess, err := mgr.Create(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to create session with auto-workspace: %v", err)
	}

	// Check that workspace ID was set
	if sess.WorkspaceID == "" {
		t.Errorf("Expected workspace ID to be set, but it was empty")
	}

	// Check that workspace was created with correct name
	ws, err := wsMgr.Get(ctx, workspace.ID(sess.WorkspaceID))
	if err != nil {
		t.Fatalf("Failed to get auto-created workspace: %v", err)
	}

	if ws.Name != sess.ID {
		t.Errorf("Expected workspace name %s, got %s", sess.ID, ws.Name)
	}

	if !ws.AutoCreated {
		t.Errorf("Expected workspace to be marked as auto-created")
	}
}

func TestManager_SendInput(t *testing.T) {
	// Create a custom runtime that supports InputSender from the start
	store := newMockStore()
	mockInputRuntime := &mockRuntimeWithInputSender{
		mockRuntime: newMockRuntime("local-input"),
	}
	mockRuntime := newMockRuntime("local")

	runtimes := map[string]runtime.Runtime{
		"local-input": mockInputRuntime,
		"local":       mockRuntime,
		"tmux":        newMockRuntime("tmux"),
	}

	taskMgr := task.NewManager()
	mgr := NewManager(store, runtimes, taskMgr, nil, nil).(*manager)
	ctx := context.Background()

	// Create a session with input-supporting runtime
	session, err := mgr.Create(ctx, CreateOptions{
		WorkspaceID: "test-workspace",
		Command:     []string{"bash"},
		Runtime:     "local-input",
	})
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Send input to the session
	testInput := "test command"
	err = mgr.SendInput(ctx, session.ID, testInput)
	if err != nil {
		t.Fatalf("Failed to send input: %v", err)
	}

	// Since we can't access the process directly anymore,
	// we just verify that SendInput didn't return an error
	// The actual input sending is tested at the runtime level

	// Test sending input to non-existent session
	err = mgr.SendInput(ctx, "non-existent", testInput)
	if err == nil {
		t.Error("Expected error for non-existent session")
	}

	// Test sending input to a stopped session
	mgr.mu.Lock()
	mgr.sessions[session.ID].Status = StatusStopped
	mgr.mu.Unlock()
	store.Save(ctx, mgr.sessions[session.ID])

	err = mgr.SendInput(ctx, session.ID, testInput)
	if err == nil {
		t.Error("Expected error for stopped session")
	}

	// Test with runtime that doesn't support InputSender
	session2, err := mgr.Create(ctx, CreateOptions{
		WorkspaceID: "test-workspace",
		Command:     []string{"echo", "test"},
		Runtime:     "local",
	})
	if err != nil {
		t.Fatalf("Failed to create session 2: %v", err)
	}

	// Don't make process2 implement InputSender
	err = mgr.SendInput(ctx, session2.ID, testInput)
	if err == nil || !contains(err.Error(), "input not supported") {
		t.Errorf("Expected 'input not supported' error, got: %v", err)
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
