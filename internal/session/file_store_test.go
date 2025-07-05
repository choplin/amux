package session

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileStore_SaveLoad(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir).(*FileStore)
	ctx := context.Background()

	// Create test session
	session := &Session{
		ID:          "test-session-1",
		WorkspaceID: "workspace-1",
		TaskName:    "test-task",
		Runtime:     "local",
		Status:      StatusRunning,
		StartedAt:   time.Now(),
		Command:     []string{"echo", "hello"},
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
		WorkingDir: "/tmp/test",
		Metadata: map[string]interface{}{
			"custom": "data",
		},
	}

	// Save session
	err := store.Save(ctx, session)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Verify file was created
	sessionFile := store.sessionFile(session.ID)
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Error("Session file was not created")
	}

	// Load session
	loaded, err := store.Load(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to load session: %v", err)
	}

	// Verify loaded data
	if loaded.ID != session.ID {
		t.Errorf("ID mismatch: expected %s, got %s", session.ID, loaded.ID)
	}
	if loaded.WorkspaceID != session.WorkspaceID {
		t.Errorf("WorkspaceID mismatch: expected %s, got %s", session.WorkspaceID, loaded.WorkspaceID)
	}
	if loaded.TaskName != session.TaskName {
		t.Errorf("TaskName mismatch: expected %s, got %s", session.TaskName, loaded.TaskName)
	}
	if loaded.Runtime != session.Runtime {
		t.Errorf("Runtime mismatch: expected %s, got %s", session.Runtime, loaded.Runtime)
	}
	if loaded.Status != session.Status {
		t.Errorf("Status mismatch: expected %s, got %s", session.Status, loaded.Status)
	}
	if len(loaded.Command) != len(session.Command) {
		t.Errorf("Command length mismatch: expected %d, got %d", len(session.Command), len(loaded.Command))
	}
	if loaded.Environment["TEST_VAR"] != session.Environment["TEST_VAR"] {
		t.Error("Environment variable mismatch")
	}
	if loaded.WorkingDir != session.WorkingDir {
		t.Errorf("WorkingDir mismatch: expected %s, got %s", session.WorkingDir, loaded.WorkingDir)
	}
}

func TestFileStore_LoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir).(*FileStore)
	ctx := context.Background()

	// Try to load non-existent session
	_, err := store.Load(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestFileStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir).(*FileStore)
	ctx := context.Background()

	// Create multiple sessions
	sessions := []*Session{
		{
			ID:          "session-1",
			WorkspaceID: "workspace-1",
			Status:      StatusRunning,
			StartedAt:   time.Now(),
			Command:     []string{"cmd1"},
		},
		{
			ID:          "session-2",
			WorkspaceID: "workspace-1",
			Status:      StatusStopped,
			StartedAt:   time.Now(),
			Command:     []string{"cmd2"},
		},
		{
			ID:          "session-3",
			WorkspaceID: "workspace-2",
			Status:      StatusRunning,
			StartedAt:   time.Now(),
			Command:     []string{"cmd3"},
		},
	}

	// Save all sessions
	for _, session := range sessions {
		if err := store.Save(ctx, session); err != nil {
			t.Fatalf("Failed to save session %s: %v", session.ID, err)
		}
	}

	// List all sessions
	listed, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(listed) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(listed))
	}

	// List sessions for workspace-1
	listed, err = store.List(ctx, "workspace-1")
	if err != nil {
		t.Fatalf("Failed to list workspace sessions: %v", err)
	}
	if len(listed) != 2 {
		t.Errorf("Expected 2 sessions for workspace-1, got %d", len(listed))
	}

	// Verify correct sessions are returned
	for _, session := range listed {
		if session.WorkspaceID != "workspace-1" {
			t.Errorf("Unexpected workspace ID: %s", session.WorkspaceID)
		}
	}

	// List sessions for workspace-2
	listed, err = store.List(ctx, "workspace-2")
	if err != nil {
		t.Fatalf("Failed to list workspace-2 sessions: %v", err)
	}
	if len(listed) != 1 {
		t.Errorf("Expected 1 session for workspace-2, got %d", len(listed))
	}
}

func TestFileStore_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir).(*FileStore)
	ctx := context.Background()

	// Create and save a session
	session := &Session{
		ID:          "test-remove",
		WorkspaceID: "workspace-1",
		Status:      StatusStopped,
		StartedAt:   time.Now(),
		Command:     []string{"echo", "test"},
	}

	err := store.Save(ctx, session)
	if err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Create a log file
	logFile := store.logFile(session.ID)
	if err := os.WriteFile(logFile, []byte("test logs"), 0o644); err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	// Remove the session
	err = store.Remove(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to remove session: %v", err)
	}

	// Verify session file is removed
	if _, err := os.Stat(store.sessionFile(session.ID)); !os.IsNotExist(err) {
		t.Error("Session file should be removed")
	}

	// Verify log file is removed
	if _, err := os.Stat(logFile); !os.IsNotExist(err) {
		t.Error("Log file should be removed")
	}

	// Try to load removed session
	_, err = store.Load(ctx, session.ID)
	if err == nil {
		t.Error("Should not be able to load removed session")
	}
}

func TestFileStore_Logs(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir).(*FileStore)
	ctx := context.Background()

	sessionID := "test-logs"
	logContent := "This is a test log\nWith multiple lines\n"

	// Create log file
	logFile := store.logFile(sessionID)
	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		t.Fatalf("Failed to create log directory: %v", err)
	}
	if err := os.WriteFile(logFile, []byte(logContent), 0o644); err != nil {
		t.Fatalf("Failed to write log file: %v", err)
	}

	// Get logs
	reader, err := store.GetLogs(ctx, sessionID)
	if err != nil {
		t.Fatalf("Failed to get logs: %v", err)
	}
	defer reader.Close()

	// Read logs
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read logs: %v", err)
	}

	if string(data) != logContent {
		t.Errorf("Log content mismatch: expected %q, got %q", logContent, string(data))
	}
}

func TestFileStore_SaveLogs(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir).(*FileStore)
	ctx := context.Background()

	sessionID := "test-save-logs"
	logContent := "Log line 1\nLog line 2\nLog line 3\n"

	// Save logs
	reader := strings.NewReader(logContent)
	err := store.SaveLogs(ctx, sessionID, reader)
	if err != nil {
		t.Fatalf("Failed to save logs: %v", err)
	}

	// Verify log file was created
	logFile := store.logFile(sessionID)
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if string(data) != logContent {
		t.Errorf("Saved log content mismatch: expected %q, got %q", logContent, string(data))
	}
}

func TestFileStore_InvalidSessionFile(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir).(*FileStore)
	ctx := context.Background()

	// Create invalid JSON file
	sessionFile := store.sessionFile("invalid-session")
	if err := os.MkdirAll(filepath.Dir(sessionFile), 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}
	if err := os.WriteFile(sessionFile, []byte("invalid json"), 0o644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	// Try to load invalid session
	_, err := store.Load(ctx, "invalid-session")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("Expected unmarshal error, got: %v", err)
	}
}

func TestFileStore_DirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	// Remove the directory to test creation
	os.RemoveAll(tmpDir)

	store := NewFileStore(tmpDir).(*FileStore)
	ctx := context.Background()

	// List should create directory if it doesn't exist
	sessions, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}

	// Verify directory was created
	if _, err := os.Stat(store.sessionDir()); os.IsNotExist(err) {
		t.Error("Session directory should have been created")
	}
}

func TestFileStore_NonSessionFiles(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir).(*FileStore)
	ctx := context.Background()

	// Create a valid session
	session := &Session{
		ID:        "valid-session",
		Status:    StatusRunning,
		StartedAt: time.Now(),
		Command:   []string{"test"},
	}
	if err := store.Save(ctx, session); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	// Create non-session files that should be ignored
	sessionDir := store.sessionDir()
	nonSessionFiles := []string{
		"random.txt",
		"session.json",          // Missing ID prefix
		"session-invalid",       // Missing .json extension
		"temp-session-123.json", // Wrong prefix
	}

	for _, filename := range nonSessionFiles {
		filePath := filepath.Join(sessionDir, filename)
		if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
			t.Fatalf("Failed to create non-session file %s: %v", filename, err)
		}
	}

	// List sessions - should only return the valid one
	sessions, err := store.List(ctx, "")
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	if sessions[0].ID != "valid-session" {
		t.Errorf("Expected session ID 'valid-session', got %s", sessions[0].ID)
	}
}
