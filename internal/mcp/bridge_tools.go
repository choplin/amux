package mcp

import (
	"context"
	"fmt"

	"github.com/aki/amux/internal/core/workspace"
	"github.com/mark3labs/mcp-go/mcp"
)

// Bridge tools provide tool-based access to MCP resources
// for clients that don't support native resource reading.
// These tools return the same data as their resource counterparts.

// promptInfo contains basic prompt information
type promptInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// registerBridgeTools registers all bridge tools that provide access to resources
func (s *ServerV2) registerBridgeTools() error {
	// Register workspace bridge tools
	if err := s.registerWorkspaceBridgeTools(); err != nil {
		return fmt.Errorf("failed to register workspace bridge tools: %w", err)
	}

	// Register prompt bridge tools
	if err := s.registerPromptBridgeTools(); err != nil {
		return fmt.Errorf("failed to register prompt bridge tools: %w", err)
	}

	// Register session bridge tools
	if err := s.registerSessionBridgeTools(); err != nil {
		return fmt.Errorf("failed to register session bridge tools: %w", err)
	}

	return nil
}

// Shared logic for getting workspace list
func (s *ServerV2) getWorkspaceList(ctx context.Context) ([]workspaceInfo, error) {
	workspaces, err := s.workspaceManager.List(ctx, workspace.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	workspaceList := make([]workspaceInfo, len(workspaces))
	for i, ws := range workspaces {
		info := workspaceInfo{
			ID:             ws.ID,
			Index:          ws.Index,
			Name:           ws.Name,
			Branch:         ws.Branch,
			BaseBranch:     ws.BaseBranch,
			Description:    ws.Description,
			StoragePath:    ws.StoragePath,
			Status:         getWorkspaceStatusString(ws),
			SemaphoreCount: ws.GetHolderCount(),
			CreatedAt:      ws.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:      ws.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		info.Resources.Detail = fmt.Sprintf("amux://workspace/%s", ws.ID)
		info.Resources.Files = fmt.Sprintf("amux://workspace/%s/files", ws.ID)
		info.Resources.Context = fmt.Sprintf("amux://workspace/%s/context", ws.ID)
		workspaceList[i] = info
	}

	return workspaceList, nil
}

// getWorkspaceStatusString returns a human-readable status string for the workspace
func getWorkspaceStatusString(ws *workspace.Workspace) string {
	// First check workspace consistency
	switch ws.Status {
	case workspace.StatusConsistent:
		// Workspace is consistent, show holder status
		switch ws.GetHolderCount() {
		case 0:
			return "available"
		case 1:
			return "held-by-1-session"
		default:
			return fmt.Sprintf("held-by-%d-sessions", ws.GetHolderCount())
		}
	case workspace.StatusFolderMissing:
		return "folder-missing"
	case workspace.StatusWorktreeMissing:
		return "worktree-missing"
	case workspace.StatusOrphaned:
		return "orphaned"
	default:
		return "unknown"
	}
}

// Shared logic for getting workspace details
func (s *ServerV2) getWorkspaceDetail(ctx context.Context, workspaceID string) (*workspaceDetail, error) {
	ws, err := s.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(workspaceID))
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	detail := &workspaceDetail{
		ID:          ws.ID,
		Index:       ws.Index,
		Name:        ws.Name,
		Branch:      ws.Branch,
		BaseBranch:  ws.BaseBranch,
		Description: ws.Description,
		Path:        ws.Path,
		CreatedAt:   ws.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   ws.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Add paths
	detail.Paths.Worktree = ws.Path
	detail.Paths.Storage = ws.StoragePath

	// Add resource URIs
	detail.Resources.Files = fmt.Sprintf("amux://workspace/%s/files", ws.ID)
	detail.Resources.Context = fmt.Sprintf("amux://workspace/%s/context", ws.ID)

	return detail, nil
}

// getRegisteredPrompts returns the list of registered prompts
func (s *ServerV2) getRegisteredPrompts() []mcp.Prompt {
	// Hard-coded list of our registered prompts
	// In the future, we might want to store these when registering
	return []mcp.Prompt{
		{
			Name:        "start-issue-work",
			Description: "Guide through starting work on a GitHub issue. Helps AI agents properly understand requirements before starting implementation",
			Arguments: []mcp.PromptArgument{
				{
					Name:        "issue_number",
					Description: "GitHub issue number to work on",
					Required:    true,
				},
				{
					Name:        "issue_title",
					Description: "Title of the GitHub issue",
					Required:    false,
				},
				{
					Name:        "issue_url",
					Description: "Full URL to the GitHub issue",
					Required:    false,
				},
			},
		},
		{
			Name:        "prepare-pr",
			Description: "Guide through preparing a pull request. Helps ensure all tests pass and code is properly formatted before creating a PR",
			Arguments: []mcp.PromptArgument{
				{
					Name:        "workspace_identifier",
					Description: "Workspace ID, index, or name to prepare PR from",
					Required:    true,
				},
				{
					Name:        "pr_title",
					Description: "Proposed PR title",
					Required:    false,
				},
				{
					Name:        "pr_description",
					Description: "Proposed PR description",
					Required:    false,
				},
			},
		},
		{
			Name:        "review-workspace",
			Description: "Analyze workspace state and suggest next steps. Helps AI agents understand what work remains to be done",
			Arguments: []mcp.PromptArgument{
				{
					Name:        "workspace_identifier",
					Description: "Workspace ID, index, or name to review",
					Required:    true,
				},
			},
		},
	}
}

// Shared logic for getting prompt list
func (s *ServerV2) getPromptList() ([]promptInfo, error) {
	// Get registered prompts
	prompts := s.getRegisteredPrompts()

	promptList := make([]promptInfo, 0, len(prompts))
	for _, prompt := range prompts {
		promptList = append(promptList, promptInfo{
			Name:        prompt.Name,
			Description: prompt.Description,
		})
	}

	return promptList, nil
}

// Shared logic for getting prompt detail
func (s *ServerV2) getPromptDetail(name string) (map[string]interface{}, error) {
	// Get registered prompts
	prompts := s.getRegisteredPrompts()

	for _, prompt := range prompts {
		if prompt.Name == name {
			detail := map[string]interface{}{
				"name":        prompt.Name,
				"description": prompt.Description,
			}

			// Add arguments if any
			if len(prompt.Arguments) > 0 {
				detail["arguments"] = prompt.Arguments
			}

			return detail, nil
		}
	}

	return nil, fmt.Errorf("prompt not found: %s", name)
}
