package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/session"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestSessionResources(t *testing.T) {
	// Create test server
	testServer := setupTestServer(t)

	t.Run("session list resource returns empty list initially", func(t *testing.T) {
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: "amux://session",
			},
		}

		results, err := testServer.handleSessionListResource(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}

		textResource, ok := results[0].(*mcp.TextResourceContents)
		if !ok {
			t.Fatalf("expected TextResourceContents, got %T", results[0])
		}

		// Parse JSON response
		var sessions []sessionInfo
		if err := json.Unmarshal([]byte(textResource.Text), &sessions); err != nil {
			t.Fatalf("failed to parse session list: %v", err)
		}

		if len(sessions) != 0 {
			t.Errorf("expected empty session list, got %d sessions", len(sessions))
		}
	})

	t.Run("session detail resource returns error for non-existent session", func(t *testing.T) {
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: "amux://session/non-existent",
			},
		}

		_, err := testServer.handleSessionDetailResource(context.Background(), req)
		if err == nil {
			t.Error("expected error for non-existent session, got nil")
		}
	})

	t.Run("session output resource returns error for non-existent session", func(t *testing.T) {
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: "amux://session/non-existent/output",
			},
		}

		_, err := testServer.handleSessionOutputResource(context.Background(), req)
		if err == nil {
			t.Error("expected error for non-existent session, got nil")
		}
	})

	t.Run("session with name and description", func(t *testing.T) {
		// Skip if tmux is not available
		tmuxAdapter, err := tmux.NewAdapter()
		if err != nil || !tmuxAdapter.IsAvailable() {
			t.Skip("tmux not available on this system")
		}

		// Create a workspace for the session
		workspaceResult, err := testServer.handleWorkspaceCreate(context.Background(), mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "workspace_create",
				Arguments: map[string]interface{}{
					"name": "test-workspace",
				},
			},
		})
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}

		// Extract workspace ID from enhanced result
		workspaceText := workspaceResult.Content[0].(mcp.TextContent).Text
		var enhancedResult struct {
			Result struct {
				ID string `json:"id"`
			} `json:"result"`
		}
		err = json.Unmarshal([]byte(workspaceText), &enhancedResult)
		if err != nil {
			t.Fatalf("failed to parse enhanced result: %v", err)
		}
		workspaceID := enhancedResult.Result.ID

		// Create session with name and description using direct manager call
		sessionManager, err := testServer.createSessionManager()
		if err != nil {
			t.Fatalf("failed to create session manager: %v", err)
		}

		opts := session.Options{
			WorkspaceID: workspaceID,
			AgentID:     "test-agent",
			Name:        "Test Session",
			Description: "A test session for verifying name and description fields",
		}

		sess, err := sessionManager.CreateSession(context.Background(), opts)
		if err != nil {
			t.Fatalf("failed to create session: %v", err)
		}

		// Get session list
		listResult, err := testServer.handleSessionListResource(context.Background(), mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: "amux://session",
			},
		})
		if err != nil {
			t.Fatalf("failed to get session list: %v", err)
		}
		if len(listResult) != 1 {
			t.Fatalf("expected 1 result, got %d", len(listResult))
		}

		// Parse response
		textResource := listResult[0].(*mcp.TextResourceContents)
		var sessions []sessionInfo
		err = json.Unmarshal([]byte(textResource.Text), &sessions)
		if err != nil {
			t.Fatalf("failed to parse sessions: %v", err)
		}
		if len(sessions) != 1 {
			t.Fatalf("expected 1 session, got %d", len(sessions))
		}

		// Verify name and description
		if sessions[0].Name != "Test Session" {
			t.Errorf("expected name 'Test Session', got '%s'", sessions[0].Name)
		}
		if sessions[0].Description != "A test session for verifying name and description fields" {
			t.Errorf("expected description 'A test session for verifying name and description fields', got '%s'", sessions[0].Description)
		}

		// Get session detail
		detailResult, err := testServer.handleSessionDetailResource(context.Background(), mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: fmt.Sprintf("amux://session/%s", sess.ID()),
			},
		})
		if err != nil {
			t.Fatalf("failed to get session detail: %v", err)
		}
		if len(detailResult) != 1 {
			t.Fatalf("expected 1 result, got %d", len(detailResult))
		}

		// Parse detail response
		detailTextResource := detailResult[0].(*mcp.TextResourceContents)
		var detail sessionDetail
		err = json.Unmarshal([]byte(detailTextResource.Text), &detail)
		if err != nil {
			t.Fatalf("failed to parse session detail: %v", err)
		}

		// Verify name and description in detail
		if detail.Name != "Test Session" {
			t.Errorf("expected name 'Test Session', got '%s'", detail.Name)
		}
		if detail.Description != "A test session for verifying name and description fields" {
			t.Errorf("expected description 'A test session for verifying name and description fields', got '%s'", detail.Description)
		}
	})
}

func TestSessionResourceRegistration(t *testing.T) {
	// Create test server
	testServer := setupTestServer(t)

	// Test that resources can be registered without error
	err := testServer.registerSessionResources()
	if err != nil {
		t.Fatalf("failed to register session resources: %v", err)
	}

	// Verify resources are registered by checking the MCP server
	// Note: This is a basic test - more comprehensive tests would require
	// creating actual sessions and verifying the resource outputs
}
