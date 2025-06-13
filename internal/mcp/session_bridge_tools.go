package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerSessionBridgeTools registers bridge tools for session resources
func (s *ServerV2) registerSessionBridgeTools() error {
	// resource_session_list - Bridge to amux://session
	listOpts, err := WithStructOptions(
		"List all active sessions (bridge to amux://session resource). Returns the same data as the session resource.",
		struct{}{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_session_list options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_session_list", listOpts...), s.handleResourceSessionList)

	// resource_session_show - Bridge to amux://session/{id}
	type SessionIDParams struct {
		SessionID string `json:"session_id" jsonschema:"required,description=Session ID or short ID"`
	}
	showOpts, err := WithStructOptions(
		"Get session details (bridge to amux://session/{id} resource). Returns the same data as the session detail resource.",
		SessionIDParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_session_show options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_session_show", showOpts...), s.handleResourceSessionShow)

	// resource_session_output - Bridge to amux://session/{id}/output
	outputOpts, err := WithStructOptions(
		"Read session output/logs (bridge to amux://session/{id}/output resource). Returns the current session output.",
		SessionIDParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_session_output options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_session_output", outputOpts...), s.handleResourceSessionOutput)

	// resource_session_mailbox - Bridge to amux://session/{id}/mailbox
	mailboxOpts, err := WithStructOptions(
		"Access session mailbox state (bridge to amux://session/{id}/mailbox resource). Returns mailbox messages and metadata.",
		SessionIDParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_session_mailbox options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_session_mailbox", mailboxOpts...), s.handleResourceSessionMailbox)

	return nil
}

// Bridge tool handlers

func (s *ServerV2) handleResourceSessionList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Create session manager
	sessionManager, err := s.createSessionManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	sessions, err := sessionManager.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessionList := make([]sessionInfo, len(sessions))
	for i, sess := range sessions {
		info := sess.Info()
		sessionInfo := sessionInfo{
			ID:          info.ID,
			Index:       info.Index,
			WorkspaceID: info.WorkspaceID,
			AgentID:     info.AgentID,
			Status:      info.StatusState.Status,
			CreatedAt:   info.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		if info.StartedAt != nil {
			sessionInfo.StartedAt = info.StartedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		if info.StoppedAt != nil {
			sessionInfo.StoppedAt = info.StoppedAt.Format("2006-01-02T15:04:05Z07:00")
		}

		// Add resource URIs
		sessionInfo.Resources.Detail = fmt.Sprintf("amux://session/%s", info.ID)
		sessionInfo.Resources.Output = fmt.Sprintf("amux://session/%s/output", info.ID)
		sessionInfo.Resources.Mailbox = fmt.Sprintf("amux://session/%s/mailbox", info.ID)

		sessionList[i] = sessionInfo
	}

	jsonData, err := json.MarshalIndent(sessionList, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session list: %w", err)
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

func (s *ServerV2) handleResourceSessionShow(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing session_id argument")
	}

	// Create a proper ReadResourceRequest
	resourceRequest := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: fmt.Sprintf("amux://session/%s", sessionID),
		},
	}

	resources, err := s.handleSessionDetailResource(ctx, resourceRequest)
	if err != nil {
		return nil, err
	}

	// Extract text from first resource
	if len(resources) == 0 {
		return nil, fmt.Errorf("no resource returned")
	}

	textResource, ok := resources[0].(*mcp.TextResourceContents)
	if !ok {
		return nil, fmt.Errorf("unexpected resource type")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: textResource.Text,
			},
		},
	}, nil
}

func (s *ServerV2) handleResourceSessionOutput(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing session_id argument")
	}

	// Create a proper ReadResourceRequest
	resourceRequest := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: fmt.Sprintf("amux://session/%s/output", sessionID),
		},
	}

	resources, err := s.handleSessionOutputResource(ctx, resourceRequest)
	if err != nil {
		return nil, err
	}

	// Extract text from first resource
	if len(resources) == 0 {
		return nil, fmt.Errorf("no resource returned")
	}

	textResource, ok := resources[0].(*mcp.TextResourceContents)
	if !ok {
		return nil, fmt.Errorf("unexpected resource type")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: textResource.Text,
			},
		},
	}, nil
}

func (s *ServerV2) handleResourceSessionMailbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	sessionID, ok := args["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing session_id argument")
	}

	// Create a proper ReadResourceRequest
	resourceRequest := mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{
			URI: fmt.Sprintf("amux://session/%s/mailbox", sessionID),
		},
	}

	resources, err := s.handleSessionMailboxResource(ctx, resourceRequest)
	if err != nil {
		return nil, err
	}

	// Extract text from first resource
	if len(resources) == 0 {
		return nil, fmt.Errorf("no resource returned")
	}

	textResource, ok := resources[0].(*mcp.TextResourceContents)
	if !ok {
		return nil, fmt.Errorf("unexpected resource type")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: textResource.Text,
			},
		},
	}, nil
}
