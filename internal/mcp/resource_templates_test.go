package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/core/workspace"
)

func TestParseWorkspaceURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		wantID   string
		wantPath string
		wantErr  bool
	}{
		{
			name:     "workspace detail URI",
			uri:      "amux://workspace/ws-123",
			wantID:   "ws-123",
			wantPath: "",
		},
		{
			name:     "workspace files root URI",
			uri:      "amux://workspace/ws-123/files",
			wantID:   "ws-123",
			wantPath: "files",
		},
		{
			name:     "workspace files with path",
			uri:      "amux://workspace/ws-123/files/src/main.go",
			wantID:   "ws-123",
			wantPath: "files/src/main.go",
		},
		{
			name:     "workspace context URI",
			uri:      "amux://workspace/ws-123/context",
			wantID:   "ws-123",
			wantPath: "context",
		},
		{
			name:    "invalid URI - no workspace prefix",
			uri:     "amux://invalid/path",
			wantErr: true,
		},
		{
			name:    "invalid URI - missing workspace ID",
			uri:     "amux://workspace/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, path, err := parseWorkspaceURI(tt.uri)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
				assert.Equal(t, tt.wantPath, path)
			}
		})
	}
}

func TestHandleWorkspaceDetailResource(t *testing.T) {
	s := setupTestServer(t)

	// Create a test workspace
	ws, err := s.workspaceManager.Create(workspace.CreateOptions{
		Name:        "test-detail",
		Description: "Test workspace for detail resource",
		BaseBranch:  "main",
	})
	require.NoError(t, err)

	ctx := context.Background()
	request := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: fmt.Sprintf("amux://workspace/%s", ws.ID),
		},
	}

	contents, err := s.handleWorkspaceDetailResource(ctx, request)
	require.NoError(t, err)
	require.Len(t, contents, 1)

	textContent, ok := contents[0].(*mcp.TextResourceContents)
	require.True(t, ok)
	assert.Equal(t, request.Params.URI, textContent.URI)
	assert.Equal(t, "application/json", textContent.MIMEType)

	// Parse the JSON to verify structure
	var detail map[string]interface{}
	err = json.Unmarshal([]byte(textContent.Text), &detail)
	require.NoError(t, err)

	assert.Equal(t, ws.ID, detail["id"])
	assert.Equal(t, "test-detail", detail["name"])
	assert.Equal(t, "Test workspace for detail resource", detail["description"])
	assert.Equal(t, "main", detail["baseBranch"])

	// Check paths are included
	paths, ok := detail["paths"].(map[string]interface{})
	require.True(t, ok, "paths field should be present")
	assert.NotEmpty(t, paths["worktree"])
	assert.NotEmpty(t, paths["context"])

	// Check resources are included
	resources, ok := detail["resources"].(map[string]interface{})
	require.True(t, ok, "resources field should be present")
	assert.Equal(t, fmt.Sprintf("amux://workspace/%s/files", ws.ID), resources["files"])
	assert.Equal(t, fmt.Sprintf("amux://workspace/%s/context", ws.ID), resources["context"])
}

func TestHandleWorkspaceFilesResource(t *testing.T) {
	s := setupTestServer(t)

	// Create a test workspace
	ws, err := s.workspaceManager.Create(workspace.CreateOptions{
		Name: "test-files",
	})
	require.NoError(t, err)

	// Create test files
	testFile := filepath.Join(ws.Path, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	testDir := filepath.Join(ws.Path, "src")
	err = os.MkdirAll(testDir, 0o755)
	require.NoError(t, err)

	t.Run("list directory", func(t *testing.T) {
		ctx := context.Background()
		request := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: fmt.Sprintf("amux://workspace/%s/files", ws.ID),
			},
		}

		contents, err := s.handleWorkspaceFilesResource(ctx, request)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(*mcp.TextResourceContents)
		require.True(t, ok)
		assert.Equal(t, "application/json", textContent.MIMEType)

		// Parse the JSON to verify structure
		var files []map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &files)
		require.NoError(t, err)

		// Should have at least test.txt and src directory
		assert.GreaterOrEqual(t, len(files), 2)

		// Find test.txt and src
		foundFile := false
		foundDir := false
		for _, f := range files {
			if f["name"] == "test.txt" {
				foundFile = true
				assert.Equal(t, "file", f["type"])
				assert.EqualValues(t, 12, f["size"]) // "test content"
			}
			if f["name"] == "src" {
				foundDir = true
				assert.Equal(t, "directory", f["type"])
			}
		}
		assert.True(t, foundFile, "test.txt not found")
		assert.True(t, foundDir, "src directory not found")
	})

	t.Run("read file", func(t *testing.T) {
		ctx := context.Background()
		request := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: fmt.Sprintf("amux://workspace/%s/files/test.txt", ws.ID),
			},
		}

		contents, err := s.handleWorkspaceFilesResource(ctx, request)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(*mcp.TextResourceContents)
		require.True(t, ok)
		assert.Equal(t, "text/plain", textContent.MIMEType)
		assert.Equal(t, "test content", textContent.Text)
	})
}

func TestHandleWorkspaceContextResource(t *testing.T) {
	s := setupTestServer(t)

	// Create a test workspace
	ws, err := s.workspaceManager.Create(workspace.CreateOptions{
		Name: "test-context",
	})
	require.NoError(t, err)

	t.Run("no context file", func(t *testing.T) {
		// Remove the CLAUDE.workspace.md file if it exists
		contextPath := filepath.Join(ws.Path, "CLAUDE.workspace.md")
		_ = os.Remove(contextPath)

		ctx := context.Background()
		request := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: fmt.Sprintf("amux://workspace/%s/context", ws.ID),
			},
		}

		contents, err := s.handleWorkspaceContextResource(ctx, request)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(*mcp.TextResourceContents)
		require.True(t, ok)
		assert.Equal(t, "text/markdown", textContent.MIMEType)
		assert.Contains(t, textContent.Text, "No CLAUDE.workspace.md file found")
	})

	t.Run("with context file", func(t *testing.T) {
		// Create CLAUDE.workspace.md
		contextContent := "# Test Context\n\nThis is a test context file."
		contextPath := filepath.Join(ws.Path, "CLAUDE.workspace.md")
		err = os.WriteFile(contextPath, []byte(contextContent), 0o644)
		require.NoError(t, err)

		ctx := context.Background()
		request := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: fmt.Sprintf("amux://workspace/%s/context", ws.ID),
			},
		}

		contents, err := s.handleWorkspaceContextResource(ctx, request)
		require.NoError(t, err)
		require.Len(t, contents, 1)

		textContent, ok := contents[0].(*mcp.TextResourceContents)
		require.True(t, ok)
		assert.Equal(t, "text/markdown", textContent.MIMEType)
		assert.Equal(t, contextContent, textContent.Text)
	})
}

func TestRegisterResourceTemplates(t *testing.T) {
	s := setupTestServer(t)

	// Test that registerResourceTemplates completes without error
	// Since MCPServer doesn't expose ListResourceTemplates method,
	// we'll test by attempting to use the handlers

	// Create a test workspace
	ws, err := s.workspaceManager.Create(workspace.CreateOptions{
		Name: "test-templates",
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Test workspace detail handler
	detailReq := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: fmt.Sprintf("amux://workspace/%s", ws.ID),
		},
	}
	contents, err := s.handleWorkspaceDetailResource(ctx, detailReq)
	require.NoError(t, err)
	assert.NotEmpty(t, contents)

	// Test workspace files handler
	filesReq := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: fmt.Sprintf("amux://workspace/%s/files", ws.ID),
		},
	}
	contents, err = s.handleWorkspaceFilesResource(ctx, filesReq)
	require.NoError(t, err)
	assert.NotEmpty(t, contents)

	// Test workspace context handler
	contextReq := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: fmt.Sprintf("amux://workspace/%s/context", ws.ID),
		},
	}
	contents, err = s.handleWorkspaceContextResource(ctx, contextReq)
	require.NoError(t, err)
	assert.NotEmpty(t, contents)
}
