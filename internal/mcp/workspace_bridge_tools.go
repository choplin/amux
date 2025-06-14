package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// WorkspaceBrowseParams contains parameters for resource_workspace_browse tool
type WorkspaceBrowseParams struct {
	WorkspaceID string `json:"workspace_identifier" jsonschema:"required,description=Workspace ID, index, or name"`
	Path        string `json:"path,omitempty" jsonschema:"description=Path within the workspace to browse (optional)"`
}

// registerWorkspaceBridgeTools registers bridge tools for workspace resources
func (s *ServerV2) registerWorkspaceBridgeTools() error {
	// resource_workspace_list - Bridge to amux://workspace
	listOpts, err := WithStructOptions(
		"List all workspaces (bridge to amux://workspace resource). Returns the same data as the workspace resource.",
		struct{}{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_workspace_list options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_workspace_list", listOpts...), s.handleResourceWorkspaceList)

	// resource_workspace_show - Bridge to amux://workspace/{id}
	showOpts, err := WithStructOptions(
		"Get details of a specific workspace (bridge to amux://workspace/{id} resource). Returns the same data as the workspace detail resource.",
		WorkspaceIDParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_workspace_show options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_workspace_show", showOpts...), s.handleResourceWorkspaceShow)

	// resource_workspace_browse - Bridge to amux://workspace/{id}/files
	browseOpts, err := WithStructOptions(
		"Browse files in a workspace (bridge to amux://workspace/{id}/files resource). Returns directory listings or file contents.",
		WorkspaceBrowseParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_workspace_browse options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_workspace_browse", browseOpts...), s.handleResourceWorkspaceBrowse)

	return nil
}

// Bridge tool handlers for workspace resources

func (s *ServerV2) handleResourceWorkspaceList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Use shared logic with resource handler
	workspaceList, err := s.getWorkspaceList()
	if err != nil {
		return nil, err
	}

	jsonData, err := json.MarshalIndent(workspaceList, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workspace list: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}

func (s *ServerV2) handleResourceWorkspaceShow(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	workspaceID, ok := args["workspace_id"].(string)
	if !ok || workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}

	// Use shared logic with resource handler
	detail, err := s.getWorkspaceDetail(workspaceID)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workspace detail: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}

func (s *ServerV2) handleResourceWorkspaceBrowse(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	workspaceID, ok := args["workspace_id"].(string)
	if !ok || workspaceID == "" {
		return nil, fmt.Errorf("workspace_id is required")
	}

	path, _ := args["path"].(string)

	// Build the URI for the resource handler
	uri := fmt.Sprintf("amux://workspace/%s/files", workspaceID)
	if path != "" {
		uri = fmt.Sprintf("%s/%s", uri, path)
	}

	// Create a resource request
	resourceReq := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: uri,
		},
	}

	// Call the resource handler
	contents, err := s.handleWorkspaceFilesResource(ctx, resourceReq)
	if err != nil {
		return nil, err
	}

	if len(contents) == 0 {
		return nil, fmt.Errorf("no content returned")
	}

	// Convert resource contents to tool result
	content := contents[0]
	switch c := content.(type) {
	case *mcp.TextResourceContents:
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: c.Text,
				},
			},
		}, nil
	case *mcp.BlobResourceContents:
		// For binary files, return base64 encoded
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Binary file (%s): base64 encoded content", c.MIMEType),
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unexpected content type")
	}
}
