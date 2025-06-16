package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerResources registers all MCP resources
func (s *ServerV2) registerResources() error {
	// Register workspace list resource
	workspaceListResource := mcp.NewResource(
		"amux://workspace",
		"Workspace List",
		mcp.WithResourceDescription("List all amux workspaces"),
		mcp.WithMIMEType("application/json"),
	)
	s.mcpServer.AddResource(workspaceListResource, s.handleWorkspaceListResource)

	return nil
}

// workspaceInfo is the common structure for workspace information
type workspaceInfo struct {
	ID          string `json:"id"`
	Index       string `json:"index"`
	Name        string `json:"name"`
	Branch      string `json:"branch"`
	BaseBranch  string `json:"baseBranch"`
	Description string `json:"description,omitempty"`
	StoragePath string `json:"storagePath,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Resources   struct {
		Detail  string `json:"detail"`
		Files   string `json:"files"`
		Context string `json:"context"`
	} `json:"resources"`
}

// handleWorkspaceListResource returns a list of all workspaces
func (s *ServerV2) handleWorkspaceListResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	workspaceList, err := s.getWorkspaceList(ctx)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.MarshalIndent(workspaceList, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workspace list: %w", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}
