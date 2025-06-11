package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerResourceTemplates registers all MCP resource templates
func (s *ServerV2) registerResourceTemplates() error {
	// Register workspace detail template
	workspaceDetailTemplate := mcp.NewResourceTemplate(
		"amux://workspace/{id}",
		"Workspace Details",
		mcp.WithTemplateDescription("Get details of a specific workspace by ID or name"),
		mcp.WithTemplateMIMEType("application/json"),
	)
	s.mcpServer.AddResourceTemplate(workspaceDetailTemplate, s.handleWorkspaceDetailResource)

	// Register workspace files template
	workspaceFilesTemplate := mcp.NewResourceTemplate(
		"amux://workspace/{id}/files{/path*}",
		"Workspace Files",
		mcp.WithTemplateDescription("Browse files in a workspace"),
		mcp.WithTemplateMIMEType("application/json"),
	)
	s.mcpServer.AddResourceTemplate(workspaceFilesTemplate, s.handleWorkspaceFilesResource)

	// Register workspace context template
	workspaceContextTemplate := mcp.NewResourceTemplate(
		"amux://workspace/{id}/context",
		"Workspace Context",
		mcp.WithTemplateDescription("Read the context.md file for a workspace"),
		mcp.WithTemplateMIMEType("text/markdown"),
	)
	s.mcpServer.AddResourceTemplate(workspaceContextTemplate, s.handleWorkspaceContextResource)

	return nil
}

// parseWorkspaceURI extracts the workspace ID from a URI like amux://workspace/{id}
func parseWorkspaceURI(uri string) (string, string, error) {
	// Remove the scheme
	path := strings.TrimPrefix(uri, "amux://")

	// Split the path
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] != "workspace" {
		return "", "", fmt.Errorf("invalid workspace URI: %s", uri)
	}

	workspaceID := parts[1]
	if workspaceID == "" {
		return "", "", fmt.Errorf("invalid workspace URI: missing workspace ID")
	}

	// Return remaining path for file resources
	remainingPath := ""
	if len(parts) > 2 {
		remainingPath = strings.Join(parts[2:], "/")
	}

	return workspaceID, remainingPath, nil
}

// handleWorkspaceDetailResource returns details for a specific workspace
func (s *ServerV2) handleWorkspaceDetailResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	workspaceID, _, err := parseWorkspaceURI(request.Params.URI)
	if err != nil {
		return nil, err
	}

	ws, err := s.workspaceManager.Get(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	// Convert to JSON-friendly format
	type workspaceDetail struct {
		ID          string `json:"id"`
		Index       string `json:"index"`
		Name        string `json:"name"`
		Branch      string `json:"branch"`
		BaseBranch  string `json:"baseBranch"`
		Path        string `json:"path"`
		Description string `json:"description,omitempty"`
		CreatedAt   string `json:"createdAt"`
		UpdatedAt   string `json:"updatedAt"`
	}

	detail := workspaceDetail{
		ID:          ws.ID,
		Index:       ws.Index,
		Name:        ws.Name,
		Branch:      ws.Branch,
		BaseBranch:  ws.BaseBranch,
		Path:        ws.Path,
		Description: ws.Description,
		CreatedAt:   ws.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   ws.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	jsonData, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workspace detail: %w", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// handleWorkspaceFilesResource lists files in a workspace directory
func (s *ServerV2) handleWorkspaceFilesResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	workspaceID, subPath, err := parseWorkspaceURI(request.Params.URI)
	if err != nil {
		return nil, err
	}

	// Remove "files" prefix if present
	if subPath == "files" {
		subPath = ""
	} else {
		subPath = strings.TrimPrefix(subPath, "files/")
	}

	ws, err := s.workspaceManager.Get(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	// Construct the full path
	fullPath := ws.Path
	if subPath != "" {
		fullPath = filepath.Join(ws.Path, subPath)
	}

	// Security check: ensure path is within workspace
	if !strings.HasPrefix(fullPath, ws.Path) {
		return nil, fmt.Errorf("access denied: path outside workspace")
	}

	// Check if path exists and is a directory
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	if !info.IsDir() {
		// If it's a file, return its contents
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		// Determine MIME type (simplified)
		mimeType := "text/plain"
		if strings.HasSuffix(fullPath, ".json") {
			mimeType = "application/json"
		} else if strings.HasSuffix(fullPath, ".md") {
			mimeType = "text/markdown"
		} else if strings.HasSuffix(fullPath, ".go") {
			mimeType = "text/x-go"
		}

		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: mimeType,
				Text:     string(content),
			},
		}, nil
	}

	// List directory contents
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	type fileInfo struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Size int64  `json:"size"`
	}

	files := make([]fileInfo, 0, len(entries))
	for _, entry := range entries {
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fileType := "file"
		if entry.IsDir() {
			fileType = "directory"
		}

		files = append(files, fileInfo{
			Name: entry.Name(),
			Type: fileType,
			Size: info.Size(),
		})
	}

	jsonData, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal file list: %w", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// handleWorkspaceContextResource returns the context.md file for a workspace
func (s *ServerV2) handleWorkspaceContextResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	workspaceID, _, err := parseWorkspaceURI(request.Params.URI)
	if err != nil {
		return nil, err
	}

	ws, err := s.workspaceManager.Get(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	// In the new structure, context.md will be at:
	// .amux/workspaces/{workspace-id}/context.md
	// But for now, it might be in the worktree
	contextPath := filepath.Join(ws.Path, "context.md")

	// Check if context.md exists
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		// Return empty content with explanation
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "text/markdown",
				Text:     "# Workspace Context\n\nNo context.md file found in this workspace.\n",
			},
		}, nil
	}

	content, err := os.ReadFile(contextPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read context.md: %w", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "text/markdown",
			Text:     string(content),
		},
	}, nil

}
