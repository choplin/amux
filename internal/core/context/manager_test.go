package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManager_Initialize(t *testing.T) {
	workspacePath := t.TempDir()
	manager := NewManager(workspacePath)

	// Initialize context
	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize context: %v", err)
	}

	// Check context directory exists
	if !manager.Exists() {
		t.Error("Context directory should exist after initialization")
	}

	// Check all template files exist
	expectedFiles := []string{
		BackgroundFile,
		PlanFile,
		WorkingLogFile,
		ResultsSummaryFile,
	}

	for _, filename := range expectedFiles {
		filePath := manager.GetFilePath(filename)
		if _, err := os.Stat(filePath); err != nil {
			t.Errorf("Expected file %s to exist: %v", filename, err)
		}
	}
}

func TestManager_Initialize_NoOverwrite(t *testing.T) {
	workspacePath := t.TempDir()
	manager := NewManager(workspacePath)

	// Initialize context
	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize context: %v", err)
	}

	// Write custom content to a file
	backgroundPath := manager.GetFilePath(BackgroundFile)
	customContent := "Custom background content"
	if err := os.WriteFile(backgroundPath, []byte(customContent), 0644); err != nil {
		t.Fatalf("Failed to write custom content: %v", err)
	}

	// Initialize again
	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to re-initialize context: %v", err)
	}

	// Check that custom content was preserved
	content, err := os.ReadFile(backgroundPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != customContent {
		t.Error("Existing file should not be overwritten")
	}
}

func TestManager_GetPaths(t *testing.T) {
	workspacePath := "/test/workspace"
	manager := NewManager(workspacePath)

	expectedContextPath := filepath.Join(workspacePath, ContextDir)
	if manager.GetContextPath() != expectedContextPath {
		t.Errorf("Expected context path %s, got %s", expectedContextPath, manager.GetContextPath())
	}

	expectedFilePath := filepath.Join(expectedContextPath, BackgroundFile)
	if manager.GetFilePath(BackgroundFile) != expectedFilePath {
		t.Errorf("Expected file path %s, got %s", expectedFilePath, manager.GetFilePath(BackgroundFile))
	}
}

func TestManager_TemplateContent(t *testing.T) {
	workspacePath := t.TempDir()
	manager := NewManager(workspacePath)

	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize context: %v", err)
	}

	// Check background template
	backgroundContent, err := os.ReadFile(manager.GetFilePath(BackgroundFile))
	if err != nil {
		t.Fatalf("Failed to read background file: %v", err)
	}
	if !strings.Contains(string(backgroundContent), "# Background") {
		t.Error("Background file should contain expected header")
	}
	if !strings.Contains(string(backgroundContent), "## Task Overview") {
		t.Error("Background file should contain task overview section")
	}

	// Check plan template
	planContent, err := os.ReadFile(manager.GetFilePath(PlanFile))
	if err != nil {
		t.Fatalf("Failed to read plan file: %v", err)
	}
	if !strings.Contains(string(planContent), "# Implementation Plan") {
		t.Error("Plan file should contain expected header")
	}
	if !strings.Contains(string(planContent), "## Approach") {
		t.Error("Plan file should contain approach section")
	}

	// Check working log template
	logContent, err := os.ReadFile(manager.GetFilePath(WorkingLogFile))
	if err != nil {
		t.Fatalf("Failed to read working log file: %v", err)
	}
	if !strings.Contains(string(logContent), "# Working Log") {
		t.Error("Working log file should contain expected header")
	}
	if !strings.Contains(string(logContent), "## Session Started:") {
		t.Error("Working log file should contain session start time")
	}

	// Check results summary template
	resultsContent, err := os.ReadFile(manager.GetFilePath(ResultsSummaryFile))
	if err != nil {
		t.Fatalf("Failed to read results summary file: %v", err)
	}
	if !strings.Contains(string(resultsContent), "# Results Summary") {
		t.Error("Results summary file should contain expected header")
	}
	if !strings.Contains(string(resultsContent), "## Overview") {
		t.Error("Results summary file should contain overview section")
	}
}

func TestManager_AppendToWorkingLog(t *testing.T) {
	workspacePath := t.TempDir()
	manager := NewManager(workspacePath)

	// Initialize context
	if err := manager.Initialize(); err != nil {
		t.Fatalf("Failed to initialize context: %v", err)
	}

	// Add entries to working log
	entries := []string{
		"Started implementation",
		"Fixed compilation error",
		"All tests passing",
	}

	for _, entry := range entries {
		if err := manager.AppendToWorkingLog(entry); err != nil {
			t.Fatalf("Failed to append to working log: %v", err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Read log content
	logContent, err := os.ReadFile(manager.GetFilePath(WorkingLogFile))
	if err != nil {
		t.Fatalf("Failed to read working log: %v", err)
	}

	// Check that all entries are present
	content := string(logContent)
	for _, entry := range entries {
		if !strings.Contains(content, entry) {
			t.Errorf("Working log should contain entry: %s", entry)
		}
	}

	// Check timestamp format
	if !strings.Contains(content, "### [") {
		t.Error("Working log entries should have timestamp prefix")
	}

}
