package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aki/amux/internal/workspace"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeparatedStorageTools(t *testing.T) {
	// Create test server with proper setup
	server := setupTestServer(t)
	ctx := context.Background()

	// Create a test workspace
	opts := workspace.CreateOptions{
		Name: "test-storage-workspace",
	}
	ws, err := server.workspaceManager.Create(ctx, opts)
	require.NoError(t, err)

	// Create storage directory
	err = os.MkdirAll(ws.StoragePath, 0o755)
	require.NoError(t, err)

	t.Run("workspace storage tools", func(t *testing.T) {
		t.Run("workspace_storage_write creates file", func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "workspace_storage_write",
					Arguments: map[string]interface{}{
						"workspace_identifier": ws.ID,
						"path":                 "test.txt",
						"content":              "Hello, World!",
					},
				},
			}

			result, err := server.handleWorkspaceStorageWrite(ctx, request)
			require.NoError(t, err)
			assert.NotNil(t, result)

			// Verify file was created
			content, err := os.ReadFile(filepath.Join(ws.StoragePath, "test.txt"))
			require.NoError(t, err)
			assert.Equal(t, "Hello, World!", string(content))
		})

		t.Run("workspace_storage_read reads file", func(t *testing.T) {
			// Create a test file first
			testFile := filepath.Join(ws.StoragePath, "read-test.txt")
			err := os.WriteFile(testFile, []byte("Test content"), 0o644)
			require.NoError(t, err)

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "workspace_storage_read",
					Arguments: map[string]interface{}{
						"workspace_identifier": ws.ID,
						"path":                 "read-test.txt",
					},
				},
			}

			result, err := server.handleWorkspaceStorageRead(ctx, request)
			require.NoError(t, err)
			assert.NotNil(t, result)

			// Check result content
			require.Len(t, result.Content, 1)
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				assert.Contains(t, textContent.Text, "Test content")
			} else {
				t.Fatal("Expected text content")
			}
		})

		t.Run("workspace_storage_list lists files", func(t *testing.T) {
			// Create some test files
			subDir := filepath.Join(ws.StoragePath, "subdir")
			err := os.MkdirAll(subDir, 0o755)
			require.NoError(t, err)

			err = os.WriteFile(filepath.Join(subDir, "file1.txt"), []byte("content1"), 0o644)
			require.NoError(t, err)
			err = os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0o644)
			require.NoError(t, err)

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "workspace_storage_list",
					Arguments: map[string]interface{}{
						"workspace_identifier": ws.ID,
						"path":                 "subdir",
					},
				},
			}

			result, err := server.handleWorkspaceStorageList(ctx, request)
			require.NoError(t, err)
			assert.NotNil(t, result)

			// Check result contains files
			require.Len(t, result.Content, 1)
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				assert.Contains(t, textContent.Text, "file1.txt")
				assert.Contains(t, textContent.Text, "file2.txt")
			}
		})
	})
}
