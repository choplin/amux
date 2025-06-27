package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

// SessionRunParams contains parameters for session_run tool
type SessionRunParams struct {
	WorkspaceID string            `json:"workspace_identifier,omitempty" jsonschema:"description=Workspace ID, index, or name. If not provided, a new workspace will be auto-created"`
	AgentID     string            `json:"agent_id" jsonschema:"required,description=Agent ID to run (e.g. 'claude' 'gpt')"`
	Name        string            `json:"name,omitempty" jsonschema:"description=Human-readable name for the session"`
	Description string            `json:"description,omitempty" jsonschema:"description=Description of the session purpose"`
	Command     string            `json:"command,omitempty" jsonschema:"description=Override the agent's default command"`
	Environment map[string]string `json:"environment,omitempty" jsonschema:"description=Additional environment variables"`
}

// SessionIDParams contains parameters for session operations
type SessionIDParams struct {
	SessionID string `json:"session_identifier" jsonschema:"required,description=Session ID, index, or name"`
}

// SessionSendInputParams contains parameters for session_send_input tool
type SessionSendInputParams struct {
	SessionID string `json:"session_identifier" jsonschema:"required,description=Session ID, index, or name"`
	Input     string `json:"input" jsonschema:"required,description=Input text to send to the session"`
}

// registerSessionTools registers session management tools
func (s *ServerV2) registerSessionTools() error {
	// session_run tool
	runOpts, err := WithStructOptions(
		GetEnhancedDescription("session_run"),
		SessionRunParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create session_run options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("session_run", runOpts...), s.handleSessionRun)

	// session_stop tool
	stopOpts, err := WithStructOptions(
		GetEnhancedDescription("session_stop"),
		SessionIDParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create session_stop options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("session_stop", stopOpts...), s.handleSessionStop)

	// session_send_input tool
	sendOpts, err := WithStructOptions(
		GetEnhancedDescription("session_send_input"),
		SessionSendInputParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create session_send_input options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("session_send_input", sendOpts...), s.handleSessionSendInput)

	return nil
}

// handleSessionRun handles the session_run tool
func (s *ServerV2) handleSessionRun(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Extract required agent ID
	agentID, ok := args["agent_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing agent_id argument")
	}

	// Build session options
	opts := session.Options{
		AgentID: agentID,
	}

	// Set workspace ID if specified
	if workspaceID, ok := args["workspace_identifier"].(string); ok && workspaceID != "" {
		opts.WorkspaceID = workspaceID
	}
	// If WorkspaceID is empty, session manager will auto-create a workspace

	// Optional name
	if name, ok := args["name"].(string); ok && name != "" {
		opts.Name = name
	}

	// Optional description
	if description, ok := args["description"].(string); ok && description != "" {
		opts.Description = description
	}

	// Optional command
	if command, ok := args["command"].(string); ok && command != "" {
		opts.Command = command
	}

	// Optional environment
	if envMap, ok := args["environment"].(map[string]interface{}); ok {
		opts.Environment = make(map[string]string)
		for k, v := range envMap {
			if strVal, ok := v.(string); ok {
				opts.Environment[k] = strVal
			}
		}
	}

	// Create session manager
	sessionManager, err := s.createSessionManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	// Create session
	sess, err := sessionManager.CreateSession(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Start session immediately
	if err := sess.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	// Get session info for response
	info := sess.Info()

	// Build response
	response := map[string]interface{}{
		"id":           info.ID,
		"index":        info.Index,
		"name":         info.Name,
		"description":  info.Description,
		"workspace_id": info.WorkspaceID,
		"agent_id":     info.AgentID,
		"status":       string(sess.Status()),
		"command":      info.Command,
		"tmux_session": info.TmuxSession,
		"created_at":   info.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if info.StartedAt != nil {
		response["started_at"] = info.StartedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	// Add attach instruction if tmux session
	if info.TmuxSession != "" {
		attachID := info.ID
		if info.Index != "" {
			attachID = info.Index
		}
		response["attach_command"] = fmt.Sprintf("tmux attach-session -t %s", info.TmuxSession)
		response["attach_amux"] = fmt.Sprintf("amux session attach %s", attachID)

		// Check if agent has autoAttach but we're in MCP context
		if agentConfig, _ := s.configManager.GetAgent(agentID); agentConfig != nil {
			if tmuxParams, err := agentConfig.GetTmuxParams(); err == nil && tmuxParams != nil && tmuxParams.AutoAttach {
				response["auto_attach_skipped"] = "Auto-attach is not available in MCP context (no TTY)"
			}
		}
	}

	// Get workspace info to check if auto-created
	if ws, err := s.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(sess.WorkspaceID())); err == nil {
		if ws.AutoCreated {
			response["workspace_auto_created"] = true
		}
	}

	// Add success message to response
	response["message"] = "Session started successfully!"

	return createEnhancedResult("session_run", response, nil)
}

// handleSessionStop handles the session_stop tool
func (s *ServerV2) handleSessionStop(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	sessionID, ok := args["session_identifier"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing session_identifier argument")
	}

	// Create session manager
	sessionManager, err := s.createSessionManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	// Get session
	sess, err := sessionManager.ResolveSession(ctx, session.Identifier(sessionID))
	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			return nil, SessionNotFoundError(sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Stop session with hook execution
	if err := sessionManager.StopSession(ctx, sess, false); err != nil {
		return nil, fmt.Errorf("failed to stop session: %w", err)
	}

	// Get updated info
	info := sess.Info()

	// Create response with session info
	response := map[string]interface{}{
		"session_id":   sessionID,
		"agent_id":     info.AgentID,
		"workspace_id": info.WorkspaceID,
		"message":      fmt.Sprintf("Session %s stopped successfully", sessionID),
	}

	return createEnhancedResult("session_stop", response, nil)
}

// handleSessionSendInput handles the session_send_input tool
func (s *ServerV2) handleSessionSendInput(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	sessionID, ok := args["session_identifier"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing session_identifier argument")
	}

	input, ok := args["input"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing input argument")
	}

	// Create session manager
	sessionManager, err := s.createSessionManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	// Get session
	sess, err := sessionManager.ResolveSession(ctx, session.Identifier(sessionID))
	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			return nil, SessionNotFoundError(sessionID)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is running
	if !sess.Status().IsRunning() {
		return nil, SessionNotRunningError(sessionID)
	}

	// Type assert to TerminalSession
	terminalSession, ok := sess.(session.TerminalSession)
	if !ok {
		return nil, fmt.Errorf("session does not support terminal operations")
	}

	// Send input
	if err := terminalSession.SendInput(ctx, input); err != nil {
		return nil, fmt.Errorf("failed to send input: %w", err)
	}

	// Create response
	response := map[string]interface{}{
		"session_id": sessionID,
		"input":      input,
		"message":    fmt.Sprintf("Input sent to session %s", sessionID),
	}

	return createEnhancedResult("session_send_input", response, nil)
}
