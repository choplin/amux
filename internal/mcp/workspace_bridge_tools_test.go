package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/aki/amux/internal/core/workspace"
)

func TestWorkspaceBridgeTools(t *testing.T) {
	testServer := setupTestServer(t)

	// Create test workspaces
	ws1Opts := workspace.CreateOptions{
		Name:        "test-workspace-1",
		Description: "First test workspace",
	}
	ws1, err := testServer.workspaceManager.Create(ws1Opts)
	if err != nil {
		t.Fatalf("failed to create workspace 1: %v", err)
	}

	ws2Opts := workspace.CreateOptions{
		Name:        "test-workspace-2",
		Description: "Second test workspace",
	}
	ws2, err := testServer.workspaceManager.Create(ws2Opts)
	if err != nil {
		t.Fatalf("failed to create workspace 2: %v", err)
	}

	t.Run("resource_workspace_list returns all workspaces", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "resource_workspace_list",
				Arguments: map[string]interface{}{},
			},
		}

		result, err := testServer.handleResourceWorkspaceList(context.Background(), req)
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

		var workspaces []map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &workspaces); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if len(workspaces) != 2 {
			t.Errorf("expected 2 workspaces, got %d", len(workspaces))
		}
	})

	t.Run("resource_workspace_show returns workspace details", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "resource_workspace_show",
				Arguments: map[string]interface{}{
					"workspace_id": ws1.ID,
				},
			},
		}

		result, err := testServer.handleResourceWorkspaceShow(context.Background(), req)
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

		var detail map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &detail); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if detail["id"] != ws1.ID {
			t.Errorf("expected workspace ID %s, got %v", ws1.ID, detail["id"])
		}
		if detail["name"] != ws1.Name {
			t.Errorf("expected workspace name %s, got %v", ws1.Name, detail["name"])
		}
	})

	t.Run("resource_workspace_browse lists directory contents", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "resource_workspace_browse",
				Arguments: map[string]interface{}{
					"workspace_id": ws2.ID,
					"path":         "",
				},
			},
		}

		result, err := testServer.handleResourceWorkspaceBrowse(context.Background(), req)
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

		// Should return directory listing
		var files []map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &files); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Should have some files/directories
		if len(files) == 0 {
			t.Error("expected at least some files in workspace root")
		}

		// Check that entries have expected fields
		for _, file := range files {
			if _, ok := file["name"]; !ok {
				t.Error("file entry missing 'name' field")
			}
			if _, ok := file["type"]; !ok {
				t.Error("file entry missing 'type' field")
			}
		}
	})
}

func TestWorkspaceBridgeToolsRegistration(t *testing.T) {
	testServer := setupTestServer(t)

	// Just verify that setupTestServer succeeds, which includes registering workspace bridge tools
	if testServer == nil {
		t.Fatal("expected server to be created with workspace bridge tools registered")
	}
}
