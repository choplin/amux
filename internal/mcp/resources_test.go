package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/core/workspace"
)

func TestHandleWorkspaceListResource(t *testing.T) {
	s := setupTestServer(t)

	// Create a test workspace
	ws, err := s.workspaceManager.Create(context.Background(), workspace.CreateOptions{
		Name:        "test-workspace",
		Description: "Test workspace for resource testing",
	})
	require.NoError(t, err)

	ctx := context.Background()
	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "amux://workspace",
		},
	}

	contents, err := s.handleWorkspaceListResource(ctx, request)
	require.NoError(t, err)
	require.Len(t, contents, 1)

	textContent, ok := contents[0].(*mcp.TextResourceContents)
	require.True(t, ok)
	assert.Equal(t, "amux://workspace", textContent.URI)
	assert.Equal(t, "application/json", textContent.MIMEType)

	// Parse the JSON to verify structure
	var workspaces []map[string]interface{}
	err = json.Unmarshal([]byte(textContent.Text), &workspaces)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(workspaces), 1)

	// Find our test workspace
	found := false
	for _, w := range workspaces {
		if w["id"] == ws.ID {
			found = true
			assert.Equal(t, "test-workspace", w["name"])
			assert.Equal(t, "Test workspace for resource testing", w["description"])

			// Check resources are included
			resources, ok := w["resources"].(map[string]interface{})
			require.True(t, ok, "resources field should be present")
			assert.Equal(t, fmt.Sprintf("amux://workspace/%s", ws.ID), resources["detail"])
			assert.Equal(t, fmt.Sprintf("amux://workspace/%s/files", ws.ID), resources["files"])
			assert.Equal(t, fmt.Sprintf("amux://workspace/%s/context", ws.ID), resources["context"])
			break
		}
	}
	assert.True(t, found, "Test workspace not found in list")
}

func TestRegisterResources(t *testing.T) {
	s := setupTestServer(t)

	// Test that registerResources completes without error
	// Since MCPServer doesn't expose ListResources method,
	// we'll test by attempting to read the resources

	// Test workspace list resource
	ctx := context.Background()
	workspaceReq := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: "amux://workspace",
		},
	}
	contents, err := s.handleWorkspaceListResource(ctx, workspaceReq)
	require.NoError(t, err)
	assert.NotEmpty(t, contents)
}
