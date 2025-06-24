package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aki/amux/internal/core/session"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerSessionBridgeTools registers bridge tools for session resources
func (s *ServerV2) registerSessionBridgeTools() error {
	// resource_session_list - Bridge to amux://session
	listOpts, err := WithStructOptions(
		GetEnhancedDescription("resource_session_list"),
		struct{}{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_session_list options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_session_list", listOpts...), s.handleResourceSessionList)

	// resource_session_show - Bridge to amux://session/{id}
	showOpts, err := WithStructOptions(
		GetEnhancedDescription("resource_session_show"),
		SessionIDParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_session_show options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_session_show", showOpts...), s.handleResourceSessionShow)

	// resource_session_output - Bridge to amux://session/{id}/output
	outputOpts, err := WithStructOptions(
		GetEnhancedDescription("resource_session_output"),
		SessionIDParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create resource_session_output options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("resource_session_output", outputOpts...), s.handleResourceSessionOutput)

	return nil
}

// Bridge tool handlers

func (s *ServerV2) handleResourceSessionList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Create session manager
	sessionManager, err := s.createSessionManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	sessions, err := sessionManager.ListSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessionList := make([]sessionInfo, len(sessions))
	for i, sess := range sessions {
		// Update status for running sessions
		if sess.Status().IsRunning() {
			// Try to update status if session supports terminal operations
			if terminalSess, ok := sess.(session.TerminalSession); ok {
				_ = terminalSess.UpdateStatus(ctx) // Ignore errors, use current status if update fails
			}
		}

		info := sess.Info()
		sessionInfo := sessionInfo{
			ID:          info.ID,
			Index:       info.Index,
			Name:        info.Name,
			Description: info.Description,
			WorkspaceID: info.WorkspaceID,
			AgentID:     info.AgentID,
			Status:      sess.Status(),
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

		sessionList[i] = sessionInfo
	}

	return createEnhancedResult("resource_session_list", sessionList, nil)
}

func (s *ServerV2) handleResourceSessionShow(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	sessionID, ok := args["session_identifier"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing session_identifier argument")
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

	// Parse the JSON to extract session details
	var sessionDetail map[string]interface{}
	if err := json.Unmarshal([]byte(textResource.Text), &sessionDetail); err != nil {
		// If parsing fails, return as plain text
		return createEnhancedResult("resource_session_show", textResource.Text, nil)
	}

	return createEnhancedResult("resource_session_show", sessionDetail, nil)
}

func (s *ServerV2) handleResourceSessionOutput(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	sessionID, ok := args["session_identifier"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing session_identifier argument")
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

	// Create structured output result
	result := map[string]interface{}{
		"session_id": sessionID,
		"output":     textResource.Text,
	}

	return createEnhancedResult("resource_session_output", result, nil)
}
