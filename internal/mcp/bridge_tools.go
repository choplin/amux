package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/aki/amux/internal/core/workspace"
)

// Bridge tools provide tool-based access to MCP resources
// for clients that don't support native resource reading.
// These tools return the same data as their resource counterparts.

// registerBridgeTools registers all bridge tools that provide access to resources
func (s *ServerV2) registerBridgeTools() error {
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
	type WorkspaceBrowseParams struct {
		WorkspaceID string `json:"workspace_id" jsonschema:"required,description=Workspace name or ID"`
		Path        string `json:"path,omitempty" jsonschema:"description=Path within the workspace to browse (optional)"`
	}
	browseOpts, err := WithStructOptions(
		"Browse files in a workspace (bridge to amux://workspace/{id}/files resource). Returns directory listings or file contents.",
		WorkspaceBrowseParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_workspace_browse options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_workspace_browse", browseOpts...), s.handleResourceWorkspaceBrowse)

	// prompt_list - List available prompts
	promptListOpts, err := WithStructOptions(
		"List all available prompts. Returns prompt names and descriptions.",
		struct{}{},
	)
	if err != nil {
		return fmt.Errorf("failed to create prompt_list options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("prompt_list", promptListOpts...), s.handlePromptList)

	// prompt_get - Get a specific prompt
	type PromptGetParams struct {
		Name string `json:"name" jsonschema:"required,description=Name of the prompt to retrieve"`
	}
	promptGetOpts, err := WithStructOptions(
		"Get a specific prompt by name. Returns the prompt definition including description and arguments.",
		PromptGetParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create prompt_get options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("prompt_get", promptGetOpts...), s.handlePromptGet)

	return nil
}

// Shared logic for getting workspace list
func (s *ServerV2) getWorkspaceList() ([]workspaceInfo, error) {
	workspaces, err := s.workspaceManager.List(workspace.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
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
			ContextPath: ws.ContextPath,
			CreatedAt:   ws.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   ws.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		info.Resources.Detail = fmt.Sprintf("amux://workspace/%s", ws.ID)
		info.Resources.Files = fmt.Sprintf("amux://workspace/%s/files", ws.ID)
		info.Resources.Context = fmt.Sprintf("amux://workspace/%s/context", ws.ID)
		workspaceList[i] = info
	}

	return workspaceList, nil
}

// Shared logic for getting workspace details
func (s *ServerV2) getWorkspaceDetail(workspaceID string) (*workspaceDetail, error) {
	ws, err := s.workspaceManager.Get(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	detail := &workspaceDetail{
		ID:          ws.ID,
		Index:       ws.Index,
		Name:        ws.Name,
		Branch:      ws.Branch,
		BaseBranch:  ws.BaseBranch,
		Path:        ws.Path,
		Description: ws.Description,
		CreatedAt:   ws.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   ws.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Paths: workspacePaths{
			Worktree: ws.Path,
			Context:  ws.ContextPath,
		},
		Resources: workspaceResources{
			Files:   fmt.Sprintf("amux://workspace/%s/files", ws.ID),
			Context: fmt.Sprintf("amux://workspace/%s/context", ws.ID),
		},
	}

	return detail, nil
}

// Bridge tool handlers

func (s *ServerV2) handleResourceWorkspaceList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	if !ok {
		return nil, fmt.Errorf("invalid or missing workspace_id argument")
	}

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
	if !ok {
		return nil, fmt.Errorf("invalid or missing workspace_id argument")
	}

	subPath := ""
	if p, ok := args["path"].(string); ok {
		subPath = p
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

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(content),
				},
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

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}

func (s *ServerV2) handlePromptList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get the list of prompts from the registered prompts
	// For now, we'll return a hardcoded list based on what we know exists
	prompts := []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}{
		{
			Name:        "workspace_planning",
			Description: "Generate a plan for implementing a feature or fixing an issue in a workspace",
		},
	}

	jsonData, err := json.MarshalIndent(prompts, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prompt list: %w", err)
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

func (s *ServerV2) handlePromptGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	promptName, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing name argument")
	}

	// For now, return the workspace planning prompt if requested
	if promptName == "workspace_planning" {
		prompt := struct {
			Name        string                 `json:"name"`
			Description string                 `json:"description"`
			Arguments   map[string]interface{} `json:"arguments"`
			Template    string                 `json:"template"`
		}{
			Name:        "workspace_planning",
			Description: "Generate a plan for implementing a feature or fixing an issue in a workspace",
			Arguments: map[string]interface{}{
				"issueNumber": map[string]string{
					"type":        "string",
					"description": "GitHub issue number",
					"required":    "false",
				},
				"issueTitle": map[string]string{
					"type":        "string",
					"description": "Title or description of the issue",
					"required":    "true",
				},
			},
			Template: `You are helping plan the implementation for: {{issueTitle}}{{#if issueNumber}} (Issue #{{issueNumber}}){{/if}}

Please create a detailed implementation plan that includes:
1. Understanding the requirements
2. Identifying affected components
3. Proposing the solution approach
4. Breaking down into specific tasks
5. Identifying potential risks or considerations`,
		}

		jsonData, err := json.MarshalIndent(prompt, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal prompt: %w", err)
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

	return nil, fmt.Errorf("prompt not found: %s", promptName)
}
