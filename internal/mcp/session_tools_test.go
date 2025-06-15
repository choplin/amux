package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

func TestSessionRun(t *testing.T) {
	// Skip if tmux not available
	tmuxAdapter, err := tmux.NewAdapter()
	if err != nil || !tmuxAdapter.IsAvailable() {
		t.Skip("tmux not available, skipping SessionRun test")
	}

	testServer := setupTestServer(t)

	// Create a workspace first
	wsOpts := workspace.CreateOptions{
		Name:        "test-workspace",
		Description: "Test workspace for session",
	}
	ws, err := testServer.workspaceManager.Create(context.Background(), wsOpts)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	t.Run("creates and starts session successfully", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "session_run",
				Arguments: map[string]interface{}{
					"workspace_identifier": ws.ID,
					"agent_id":             "test-agent",
					"command":              "echo 'test'",
					"environment": map[string]interface{}{
						"TEST_VAR": "test_value",
					},
				},
			},
		}

		result, err := testServer.handleSessionRun(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content, got %d", len(result.Content))
		}

		// Check response contains success message
		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", result.Content[0])
		}

		if !contains(textContent.Text, "Session started successfully") {
			t.Errorf("expected success message in response, got: %s", textContent.Text)
		}

		// Parse JSON response
		jsonStart := findJSONStart(textContent.Text)
		if jsonStart == -1 {
			t.Fatalf("no JSON found in response: %s", textContent.Text)
		}

		var response map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text[jsonStart:]), &response); err != nil {
			t.Fatalf("failed to parse response JSON: %v", err)
		}

		// Verify response fields
		if response["workspace_id"] != ws.ID {
			t.Errorf("expected workspace_id %s, got %v", ws.ID, response["workspace_id"])
		}
		if response["agent_id"] != "test-agent" {
			t.Errorf("expected agent_id test-agent, got %v", response["agent_id"])
		}
		status, ok := response["status"].(string)
		if !ok {
			t.Errorf("expected status to be string, got %T", response["status"])
		} else if status != string(session.StatusWorking) && status != string(session.StatusIdle) {
			t.Errorf("expected status to be working or idle, got %v", status)
		}

		// Clean up - stop the session
		sessionID := response["id"].(string)
		sessionManager, _ := testServer.createSessionManager()
		sess, _ := sessionManager.ResolveSession(context.Background(), session.Identifier(sessionID))
		if sess != nil {
			_ = sess.Stop()
		}
	})

	t.Run("creates session with name and description", func(t *testing.T) {
		// Use different workspace to avoid conflicts
		wsOpts2 := workspace.CreateOptions{
			Name:        "test-workspace-2",
			Description: "Test workspace for named session",
		}
		ws2, err := testServer.workspaceManager.Create(context.Background(), wsOpts2)
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "session_run",
				Arguments: map[string]interface{}{
					"workspace_identifier": ws2.ID,
					"agent_id":             "test-agent",
					"name":                 "Named Test Session",
					"description":          "Session with custom name and description",
					"command":              "sleep 1",
				},
			},
		}

		result, err := testServer.handleSessionRun(context.Background(), req)
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}

		// Extract response from text content
		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", result.Content[0])
		}

		// Parse JSON response
		text := textContent.Text
		jsonStart := strings.Index(text, "{")
		if jsonStart == -1 {
			t.Fatalf("no JSON found in response: %s", text)
		}
		var response map[string]interface{}
		if err := json.Unmarshal([]byte(text[jsonStart:]), &response); err != nil {
			t.Fatalf("failed to parse response JSON: %v", err)
		}

		// Verify name and description
		if response["name"] != "Named Test Session" {
			t.Errorf("expected name 'Named Test Session', got %v", response["name"])
		}
		if response["description"] != "Session with custom name and description" {
			t.Errorf("expected description 'Session with custom name and description', got %v", response["description"])
		}

		// Clean up - stop the session
		sessionID := response["id"].(string)
		sessionManager, _ := testServer.createSessionManager()
		sess, _ := sessionManager.ResolveSession(context.Background(), session.Identifier(sessionID))
		if sess != nil {
			_ = sess.Stop()
		}
	})

	t.Run("fails with invalid workspace", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "session_run",
				Arguments: map[string]interface{}{
					"workspace_id": "non-existent",
					"agent_id":     "test-agent",
				},
			},
		}

		_, err := testServer.handleSessionRun(context.Background(), req)
		if err == nil {
			t.Error("expected error for non-existent workspace, got nil")
		}
	})

	t.Run("fails with missing required parameters", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "session_run",
				Arguments: map[string]interface{}{
					"workspace_identifier": ws.ID,
					// missing agent_id
				},
			},
		}

		_, err := testServer.handleSessionRun(context.Background(), req)
		if err == nil {
			t.Error("expected error for missing agent_id, got nil")
		}
	})
}

func TestSessionStop(t *testing.T) {
	// Skip if tmux not available
	tmuxAdapter, err := tmux.NewAdapter()
	if err != nil || !tmuxAdapter.IsAvailable() {
		t.Skip("tmux not available, skipping SessionStop test")
	}

	testServer := setupTestServer(t)

	// Create workspace and session
	wsOpts := workspace.CreateOptions{
		Name: "test-workspace",
	}
	ws, err := testServer.workspaceManager.Create(context.Background(), wsOpts)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Create session manager and start a session
	sessionManager, err := testServer.createSessionManager()
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}

	sess, err := sessionManager.CreateSession(context.Background(), session.Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := sess.Start(context.Background()); err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	t.Run("stops running session successfully", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "session_stop",
				Arguments: map[string]interface{}{
					"session_identifier": sess.ID(),
				},
			},
		}

		result, err := testServer.handleSessionStop(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content, got %d", len(result.Content))
		}

		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", result.Content[0])
		}

		if !contains(textContent.Text, "stopped successfully") {
			t.Errorf("expected success message in response, got: %s", textContent.Text)
		}
	})

	t.Run("fails with non-existent session", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "session_stop",
				Arguments: map[string]interface{}{
					"session_identifier": "non-existent",
				},
			},
		}

		_, err := testServer.handleSessionStop(context.Background(), req)
		if err == nil {
			t.Error("expected error for non-existent session, got nil")
		}
	})
}

func TestSessionSendInput(t *testing.T) {
	// Skip if tmux not available
	tmuxAdapter, err := tmux.NewAdapter()
	if err != nil || !tmuxAdapter.IsAvailable() {
		t.Skip("tmux not available, skipping SendInput test")
	}

	testServer := setupTestServer(t)

	// Create workspace and session
	wsOpts := workspace.CreateOptions{
		Name: "test-workspace",
	}
	ws, err := testServer.workspaceManager.Create(context.Background(), wsOpts)
	if err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Create session manager and start a session
	sessionManager, err := testServer.createSessionManager()
	if err != nil {
		t.Fatalf("failed to create session manager: %v", err)
	}

	sess, err := sessionManager.CreateSession(context.Background(), session.Options{
		WorkspaceID: ws.ID,
		AgentID:     "test-agent",
	})
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := sess.Start(context.Background()); err != nil {
		t.Fatalf("failed to start session: %v", err)
	}

	t.Run("sends input to running session", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "session_send_input",
				Arguments: map[string]interface{}{
					"session_identifier": sess.ID(),
					"input":              "test input",
				},
			},
		}

		result, err := testServer.handleSessionSendInput(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content, got %d", len(result.Content))
		}

		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", result.Content[0])
		}

		if !contains(textContent.Text, "Input sent to session") {
			t.Errorf("expected success message in response, got: %s", textContent.Text)
		}
	})

	t.Run("fails with stopped session", func(t *testing.T) {
		// Stop the session first
		if err := sess.Stop(); err != nil {
			t.Fatalf("failed to stop session: %v", err)
		}

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "session_send_input",
				Arguments: map[string]interface{}{
					"session_identifier": sess.ID(),
					"input":              "test input",
				},
			},
		}

		_, err := testServer.handleSessionSendInput(context.Background(), req)
		if err == nil {
			t.Error("expected error for stopped session, got nil")
		}
		if !contains(err.Error(), "not running") {
			t.Errorf("expected 'not running' error, got: %v", err)
		}
	})
}

func TestSessionToolsRegistration(t *testing.T) {
	// Just verify that setupTestServer succeeds, which includes registering session tools
	testServer := setupTestServer(t)

	// If we got here without error, tools were registered successfully
	if testServer == nil {
		t.Fatal("expected server to be created with session tools registered")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && strings.Contains(s, substr))
}

// Helper function to find where JSON starts in response
func findJSONStart(s string) int {
	for i, ch := range s {
		if ch == '{' || ch == '[' {
			return i
		}
	}
	return -1
}
