package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBridgeTools(t *testing.T) {
	// Setup test environment
	server := setupTestServer(t)
	ctx := context.Background()

	t.Run("resource_workspace_list returns empty list initially", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "resource_workspace_list",
				Arguments: map[string]interface{}{},
			},
		}

		result, err := server.handleResourceWorkspaceList(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Content, 1)

		// Parse JSON response
		var workspaces []workspaceInfo
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &workspaces)
		require.NoError(t, err)
		assert.Empty(t, workspaces)
	})

	t.Run("resource_workspace_show returns error for non-existent workspace", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "resource_workspace_show",
				Arguments: map[string]interface{}{
					"workspace_id": "non-existent",
				},
			},
		}

		_, err := server.handleResourceWorkspaceShow(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace not found")
	})

	t.Run("resource_workspace_browse returns error for non-existent workspace", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "resource_workspace_browse",
				Arguments: map[string]interface{}{
					"workspace_id": "non-existent",
				},
			},
		}

		_, err := server.handleResourceWorkspaceBrowse(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "workspace not found")
	})

	t.Run("prompt_list returns available prompts", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "prompt_list",
				Arguments: map[string]interface{}{},
			},
		}

		result, err := server.handlePromptList(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Content, 1)

		// Parse JSON response
		var prompts []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &prompts)
		require.NoError(t, err)
		assert.NotEmpty(t, prompts)
		assert.Equal(t, "workspace_planning", prompts[0].Name)
	})

	t.Run("prompt_get returns specific prompt", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "prompt_get",
				Arguments: map[string]interface{}{
					"name": "workspace_planning",
				},
			},
		}

		result, err := server.handlePromptGet(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Content, 1)

		// Parse JSON response
		var prompt map[string]interface{}
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &prompt)
		require.NoError(t, err)
		assert.Equal(t, "workspace_planning", prompt["name"])
		assert.NotEmpty(t, prompt["template"])
	})

	t.Run("prompt_get returns error for non-existent prompt", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "prompt_get",
				Arguments: map[string]interface{}{
					"name": "non-existent",
				},
			},
		}

		_, err := server.handlePromptGet(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prompt not found")
	})
}

func TestBridgeToolsWithWorkspace(t *testing.T) {
	// Setup test environment with a workspace
	server := setupTestServer(t)
	ctx := context.Background()

	// Create a test workspace
	createRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "workspace_create",
			Arguments: map[string]interface{}{
				"name":        "test-workspace",
				"description": "Test workspace for bridge tools",
			},
		},
	}

	createResult, err := server.handleWorkspaceCreate(ctx, createRequest)
	require.NoError(t, err)
	assert.NotNil(t, createResult)

	// Parse workspace info
	var workspace map[string]interface{}
	err = json.Unmarshal([]byte(createResult.Content[0].(mcp.TextContent).Text), &workspace)
	require.NoError(t, err)
	workspaceID := workspace["id"].(string)

	t.Run("resource_workspace_list includes created workspace", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "resource_workspace_list",
				Arguments: map[string]interface{}{},
			},
		}

		result, err := server.handleResourceWorkspaceList(ctx, request)
		require.NoError(t, err)

		var workspaces []workspaceInfo
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &workspaces)
		require.NoError(t, err)
		assert.Len(t, workspaces, 1)
		assert.Equal(t, "test-workspace", workspaces[0].Name)
	})

	t.Run("resource_workspace_show returns workspace details", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "resource_workspace_show",
				Arguments: map[string]interface{}{
					"workspace_id": workspaceID,
				},
			},
		}

		result, err := server.handleResourceWorkspaceShow(ctx, request)
		require.NoError(t, err)

		var detail workspaceDetail
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &detail)
		require.NoError(t, err)
		assert.Equal(t, "test-workspace", detail.Name)
		assert.Equal(t, workspaceID, detail.ID)
		assert.NotEmpty(t, detail.Path)
		assert.NotEmpty(t, detail.Paths.Worktree)
	})

	t.Run("resource_workspace_browse lists workspace root", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "resource_workspace_browse",
				Arguments: map[string]interface{}{
					"workspace_id": workspaceID,
				},
			},
		}

		result, err := server.handleResourceWorkspaceBrowse(ctx, request)
		require.NoError(t, err)

		var files []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		}
		err = json.Unmarshal([]byte(result.Content[0].(mcp.TextContent).Text), &files)
		require.NoError(t, err)
		// Should have at least README.md from the test repo
		assert.NotEmpty(t, files)
	})

	// Cleanup
	t.Cleanup(func() {
		_ = server.workspaceManager.Remove(workspaceID)
	})
}
