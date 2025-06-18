package mcp

import (
	"context"
	"fmt"
	"strings"

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
		GetEnhancedDescription("resource_workspace_list"),
		struct{}{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_workspace_list options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_workspace_list", listOpts...), s.handleResourceWorkspaceList)

	// resource_workspace_show - Bridge to amux://workspace/{id}
	showOpts, err := WithStructOptions(
		GetEnhancedDescription("resource_workspace_show"),
		WorkspaceIDParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_workspace_show options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_workspace_show", showOpts...), s.handleResourceWorkspaceShow)

	// resource_workspace_browse - Bridge to amux://workspace/{id}/files
	// Disabled for v0.1.0: AI agents overuse it, low value for own workspace, error-prone
	// See https://github.com/choplin/amux/issues/164 for context and re-enablement considerations
	// browseOpts, err := WithStructOptions(
	// 	GetEnhancedDescription("resource_workspace_browse"),
	// 	WorkspaceBrowseParams{},
	// )
	// if err != nil {
	// 	return fmt.Errorf("failed to create resource_workspace_browse options: %w", err)
	// }
	// s.mcpServer.AddTool(mcp.NewTool("resource_workspace_browse", browseOpts...), s.handleResourceWorkspaceBrowse)

	return nil
}

// Bridge tool handlers for workspace resources

func (s *ServerV2) handleResourceWorkspaceList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Use shared logic with resource handler
	workspaceList, err := s.getWorkspaceList(ctx)
	if err != nil {
		return nil, err
	}

	return createEnhancedResult("resource_workspace_list", workspaceList, nil)
}

func (s *ServerV2) handleResourceWorkspaceShow(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	workspaceID, ok := args["workspace_identifier"].(string)
	if !ok || workspaceID == "" {
		return nil, fmt.Errorf("workspace_identifier is required")
	}

	// Use shared logic with resource handler
	detail, err := s.getWorkspaceDetail(ctx, workspaceID)
	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			return nil, WorkspaceNotFoundError(workspaceID)
		}
		return nil, err
	}

	return createEnhancedResult("resource_workspace_show", detail, nil)
}

func (s *ServerV2) handleResourceWorkspaceBrowse(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	workspaceID, ok := args["workspace_identifier"].(string)
	if !ok || workspaceID == "" {
		return nil, fmt.Errorf("workspace_identifier is required")
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
		// Create structured result for directory listings or file contents
		result := map[string]interface{}{
			"workspace_identifier": workspaceID,
			"path":                 path,
			"content":              c.Text,
		}
		return createEnhancedResult("resource_workspace_browse", result, nil)
	case *mcp.BlobResourceContents:
		// For binary files, return structured result
		result := map[string]interface{}{
			"workspace_identifier": workspaceID,
			"path":                 path,
			"type":                 "binary",
			"mime_type":            c.MIMEType,
			"message":              fmt.Sprintf("Binary file (%s)", c.MIMEType),
		}
		return createEnhancedResult("resource_workspace_browse", result, nil)
	default:
		return nil, fmt.Errorf("unexpected content type")
	}
}
