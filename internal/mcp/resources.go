package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/aki/amux/internal/core/workspace"
)

// ConventionsData represents the amux conventions exposed via MCP resource
type ConventionsData struct {
	Paths    ConventionPaths    `json:"paths"`
	Patterns ConventionPatterns `json:"patterns"`
}

// ConventionPaths defines where amux stores various components
type ConventionPaths struct {
	WorkspaceRoot     string `json:"workspace_root"`
	WorkspaceContext  string `json:"workspace_context"`
	WorkspaceMetadata string `json:"workspace_metadata"`
	SessionMailbox    string `json:"session_mailbox"`
	MailboxInbox      string `json:"mailbox_inbox"`
	MailboxOutbox     string `json:"mailbox_outbox"`
}

// ConventionPatterns defines naming patterns used by amux
type ConventionPatterns struct {
	BranchName  string `json:"branch_name"`
	WorkspaceID string `json:"workspace_id"`
	SessionID   string `json:"session_id"`
}

// registerResources registers all MCP resources
func (s *ServerV2) registerResources() error {
	// Register conventions resource
	conventionsResource := mcp.NewResource(
		"amux://conventions",
		"Amux Conventions",
		mcp.WithResourceDescription("Amux directory structure, naming patterns, and other conventions"),
		mcp.WithMIMEType("application/json"),
	)

	s.mcpServer.AddResource(conventionsResource, s.handleConventionsResource)

	// Register workspace list resource
	workspaceListResource := mcp.NewResource(
		"amux://workspace",
		"Workspace List",
		mcp.WithResourceDescription("List all amux workspaces"),
		mcp.WithMIMEType("application/json"),
	)
	s.mcpServer.AddResource(workspaceListResource, s.handleWorkspaceListResource)

	// TODO: Add more resources:
	// - amux://workspace/{id} (get details)
	// - amux://workspace/{id}/files[/{path}] (browse files)
	// - amux://workspace/{id}/context (read context)

	return nil
}

// handleConventionsResource returns the amux conventions
func (s *ServerV2) handleConventionsResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	conventions := ConventionsData{
		Paths: ConventionPaths{
			WorkspaceRoot:     ".amux/workspaces/{workspace-id}/worktree/",
			WorkspaceContext:  ".amux/workspaces/{workspace-id}/context.md",
			WorkspaceMetadata: ".amux/workspaces/{workspace-id}/metadata.json",
			SessionMailbox:    ".amux/mailbox/{session-id}/",
			MailboxInbox:      ".amux/mailbox/{session-id}/in/",
			MailboxOutbox:     ".amux/mailbox/{session-id}/out/",
		},
		Patterns: ConventionPatterns{
			BranchName:  "amux/workspace-{name}-{timestamp}-{hash}",
			WorkspaceID: "workspace-{name}-{timestamp}-{hash}",
			SessionID:   "session-{index}",
		},
	}

	jsonData, err := json.MarshalIndent(conventions, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal conventions: %w", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
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
	}

	workspaceList := make([]workspaceInfo, len(workspaces))
	for i, ws := range workspaces {
		workspaceList[i] = workspaceInfo{
			ID:          ws.ID,
			Index:       ws.Index,
			Name:        ws.Name,
			Branch:      ws.Branch,
			BaseBranch:  ws.BaseBranch,
			Description: ws.Description,
			CreatedAt:   ws.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   ws.UpdatedAt.Format(time.RFC3339),
		}
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
