package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
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

	t.Run("session storage tools", func(t *testing.T) {
		// Skip if tmux not available
		tmuxAdapter, err := tmux.NewAdapter()
		if err != nil || !tmuxAdapter.IsAvailable() {
			t.Skip("tmux not available, skipping session storage tools test")
		}

		// Create test session
		sessionManager, err := server.createSessionManager()
		require.NoError(t, err)

		opts := session.Options{
			AgentID:     "test-agent",
			WorkspaceID: ws.ID,
			Name:        "test-session",
		}

		sess, err := sessionManager.CreateSession(ctx, opts)
		require.NoError(t, err)

		// Start the session
		err = sess.Start(ctx)
		require.NoError(t, err)

		// Get session info to find storage path
		sessInfo := sess.Info()

		// Create session storage directory
		err = os.MkdirAll(sessInfo.StoragePath, 0o755)
		require.NoError(t, err)

		t.Run("session_storage_write creates file", func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "session_storage_write",
					Arguments: map[string]interface{}{
						"session_identifier": sess.ID(),
						"path":               "session-test.txt",
						"content":            "Session data",
					},
				},
			}

			result, err := server.handleSessionStorageWrite(ctx, request)
			require.NoError(t, err)
			assert.NotNil(t, result)

			// Verify file was created
			content, err := os.ReadFile(filepath.Join(sessInfo.StoragePath, "session-test.txt"))
			require.NoError(t, err)
			assert.Equal(t, "Session data", string(content))
		})

		t.Run("session_storage_read reads file", func(t *testing.T) {
			// Create a test file first
			testFile := filepath.Join(sessInfo.StoragePath, "read-test.txt")
			err := os.WriteFile(testFile, []byte("Session content"), 0o644)
			require.NoError(t, err)

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "session_storage_read",
					Arguments: map[string]interface{}{
						"session_identifier": sess.ID(),
						"path":               "read-test.txt",
					},
				},
			}

			result, err := server.handleSessionStorageRead(ctx, request)
			require.NoError(t, err)
			assert.NotNil(t, result)

			// Check result content
			require.Len(t, result.Content, 1)
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				assert.Contains(t, textContent.Text, "Session content")
			}
		})

		t.Run("session_storage_list lists files", func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "session_storage_list",
					Arguments: map[string]interface{}{
						"session_identifier": sess.ID(),
					},
				},
			}

			result, err := server.handleSessionStorageList(ctx, request)
			require.NoError(t, err)
			assert.NotNil(t, result)
		})

		// Cleanup session
		err = sess.Stop()
		require.NoError(t, err)
	})
}
