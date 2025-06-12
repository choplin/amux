package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/aki/amux/internal/core/workspace"
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

// handleWorkspaceListResource returns a list of all workspaces
func (s *ServerV2) handleWorkspaceListResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	workspaces, err := s.workspaceManager.List(workspace.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	// Convert workspaces to a simpler format for JSON
	type workspaceInfo struct {
		ID          string `json:"id"`
		Index       string `json:"index"`
		Name        string `json:"name"`
		Branch      string `json:"branch"`
		BaseBranch  string `json:"baseBranch"`
		Description string `json:"description,omitempty"`
		CreatedAt   string `json:"createdAt"`
		UpdatedAt   string `json:"updatedAt"`
		Resources   struct {
			Detail  string `json:"detail"`
			Files   string `json:"files"`
			Context string `json:"context"`
		} `json:"resources"`
	}

	workspaceList := make([]workspaceInfo, len(workspaces))
	for i, ws := range workspaces {
		info := workspaceInfo{
			ID:          ws.ID,
			Index:       ws.Index,
			Name:        ws.Name,
			Branch:      ws.Branch,
			BaseBranch:  ws.BaseBranch,
			Description: ws.Description,
			CreatedAt:   ws.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   ws.UpdatedAt.Format(time.RFC3339),
		}
		info.Resources.Detail = fmt.Sprintf("amux://workspace/%s", ws.ID)
		info.Resources.Files = fmt.Sprintf("amux://workspace/%s/files", ws.ID)
		info.Resources.Context = fmt.Sprintf("amux://workspace/%s/context", ws.ID)
		workspaceList[i] = info
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
