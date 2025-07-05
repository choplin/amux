package session

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStore_GetLogs_NewFormat(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir)

	ctx := context.Background()
	sessionID := "test-session"

	// Create session directory structure
	sessionDir := filepath.Join(tmpDir, "sessions", sessionID)

	// Create multiple run directories with logs
	runs := []struct {
		runID   string
		content string
	}{
		{"1", "First run\n"},
		{"2", "Second run\n"},
		{"3", "Third run\n"},
	}

	for _, run := range runs {
		runDir := filepath.Join(sessionDir, run.runID)
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			t.Fatal(err)
		}

		logFile := filepath.Join(runDir, "console.log")
		if err := os.WriteFile(logFile, []byte(run.content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Get logs
	reader, err := store.GetLogs(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	defer reader.Close()

	// Read all logs
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read logs: %v", err)
	}

	// Verify content (should be in order)
	expected := "First run\nSecond run\nThird run\n"
	if string(data) != expected {
		t.Errorf("Expected logs:\n%s\nGot:\n%s", expected, string(data))
	}
}

func TestFileStore_GetLogs_OldFormat(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir)

	ctx := context.Background()
	sessionID := "test-session"

	// Create old format log file
	sessionDir := filepath.Join(tmpDir, "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatal(err)
	}

	oldLogFile := filepath.Join(sessionDir, "session-"+sessionID+".log")
	content := "Old format log content\n"
	if err := os.WriteFile(oldLogFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// Get logs
	reader, err := store.GetLogs(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	defer reader.Close()

	// Read all logs
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read logs: %v", err)
	}

	// Verify content
	if string(data) != content {
		t.Errorf("Expected logs:\n%s\nGot:\n%s", content, string(data))
	}
}

func TestFileStore_GetLogs_NoLogs(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir)

	ctx := context.Background()
	sessionID := "nonexistent"

	// Get logs for non-existent session
	_, err := store.GetLogs(ctx, sessionID)
	if err == nil {
		t.Error("Expected error for non-existent session, got nil")
	}
}

func TestFileStore_GetLogs_MixedRunIDs(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	store := NewFileStore(tmpDir)

	ctx := context.Background()
	sessionID := "test-session"

	// Create session directory structure with non-sequential run IDs
	sessionDir := filepath.Join(tmpDir, "sessions", sessionID)

	// Create run directories in wrong order
	runs := []struct {
		runID   string
		content string
	}{
		{"10", "Run 10\n"},
		{"2", "Run 2\n"},
		{"1", "Run 1\n"},
		{"21", "Run 21\n"},
	}

	for _, run := range runs {
		runDir := filepath.Join(sessionDir, run.runID)
		if err := os.MkdirAll(runDir, 0o755); err != nil {
			t.Fatal(err)
		}

		logFile := filepath.Join(runDir, "console.log")
		if err := os.WriteFile(logFile, []byte(run.content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Get logs
	reader, err := store.GetLogs(ctx, sessionID)
	if err != nil {
		t.Fatalf("GetLogs failed: %v", err)
	}
	defer reader.Close()

	// Read all logs
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Failed to read logs: %v", err)
	}

	// Verify content is sorted by run ID
	expected := "Run 1\nRun 2\nRun 10\nRun 21\n"
	if string(data) != expected {
		t.Errorf("Expected logs in order:\n%s\nGot:\n%s", expected, string(data))
	}
}
