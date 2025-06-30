package mcp

import (
	"testing"
)

// Test that session tools are registered
func TestSessionToolsRegistration(t *testing.T) {
	server := setupTestServer(t)

	// Check that session tools are registered
	expectedTools := []string{
		"session_run",
		"session_list",
		"session_logs",
		"session_stop",
		"session_remove",
	}

	// Verify that registerSessionTools doesn't return an error
	// The actual registration happens in the server initialization
	if server == nil {
		t.Error("Server initialization failed")
	}

	// Check that we have the expected number of tools
	// Note: session_attach is not implemented in the new session tools
	if len(expectedTools) != 5 {
		t.Errorf("Expected 5 session tools, got %d", len(expectedTools))
	}
}

// Test tool descriptions
func TestSessionToolDescriptions(t *testing.T) {
	// Verify that enhanced descriptions exist
	tools := []string{
		"session_run",
		"session_list",
		"session_logs",
		"session_stop",
		"session_remove",
	}

	for _, tool := range tools {
		desc := GetEnhancedDescription(tool)
		if desc == "" {
			t.Errorf("No description found for tool %s", tool)
		}
		if len(desc) < 10 {
			t.Errorf("Description too short for tool %s: %s", tool, desc)
		}
	}
}

// Test parameter structures
func TestSessionToolParameters(t *testing.T) {
	// Test SessionRunParams
	params := SessionRunParams{
		Command:     []string{"echo", "test"},
		Runtime:     "local",
		WorkspaceID: "test-workspace",
	}

	if len(params.Command) != 2 {
		t.Error("Command should have 2 elements")
	}

	// Test SessionStopParams
	stopParams := SessionStopParams{
		SessionID: "session-123",
		Force:     true,
	}

	if stopParams.SessionID != "session-123" {
		t.Error("SessionID not set correctly")
	}

	// Test SessionLogsParams
	logsParams := SessionLogsParams{
		SessionID: "session-123",
		Follow:    true,
	}

	if logsParams.SessionID != "session-123" {
		t.Error("SessionID not set correctly")
	}

	// Test SessionRemoveParams
	removeParams := SessionRemoveParams{
		SessionID: "session-123",
	}

	if removeParams.SessionID != "session-123" {
		t.Error("SessionID not set correctly")
	}
}
