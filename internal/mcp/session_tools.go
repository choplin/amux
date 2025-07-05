package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/session"
	"github.com/aki/amux/internal/task"
)

// SessionRunParams defines parameters for session_run tool
type SessionRunParams struct {
	WorkspaceID         string            `json:"workspace_id,omitempty" jsonschema:"description=Workspace ID to run the session in"`
	AutoCreateWorkspace bool              `json:"auto_create_workspace,omitempty" jsonschema:"description=Auto-create workspace if not specified,default=true"`
	Name                string            `json:"name,omitempty" jsonschema:"description=Human-readable name for the session"`
	Description         string            `json:"description,omitempty" jsonschema:"description=Description of session purpose"`
	TaskName            string            `json:"task_name,omitempty" jsonschema:"description=Name of a predefined task to run"`
	Command             []string          `json:"command,omitempty" jsonschema:"description=Command and arguments to run (if no task specified)"`
	Runtime             string            `json:"runtime,omitempty" jsonschema:"description=Runtime to use (local, tmux),default=local"`
	Environment         map[string]string `json:"environment,omitempty" jsonschema:"description=Additional environment variables"`
	WorkingDir          string            `json:"working_dir,omitempty" jsonschema:"description=Working directory override"`
	EnableLog           bool              `json:"enable_log,omitempty" jsonschema:"description=Enable logging to file,default=false"`
}

// SessionListParams defines parameters for session_list tool
type SessionListParams struct {
	WorkspaceID string `json:"workspace_id,omitempty" jsonschema:"description=Filter by workspace ID"`
}

// SessionStopParams defines parameters for session_stop tool
type SessionStopParams struct {
	SessionID string `json:"session_id" jsonschema:"description=Session ID to stop,required"`
	Force     bool   `json:"force,omitempty" jsonschema:"description=Force kill the session,default=false"`
}

// SessionLogsParams defines parameters for session_logs tool
type SessionLogsParams struct {
	SessionID string `json:"session_id" jsonschema:"description=Session ID to get logs from,required"`
	Follow    bool   `json:"follow,omitempty" jsonschema:"description=Follow log output,default=false"`
}

// SessionRemoveParams defines parameters for session_remove tool
type SessionRemoveParams struct {
	SessionID     string `json:"session_id" jsonschema:"description=Session ID to remove,required"`
	KeepWorkspace bool   `json:"keep_workspace,omitempty" jsonschema:"description=Keep auto-created workspace when removing session,default=false"`
	Force         bool   `json:"force,omitempty" jsonschema:"description=Force removal by stopping running sessions first,default=false"`
}

// SessionSendKeysParams defines parameters for session_send_keys tool
type SessionSendKeysParams struct {
	SessionID string `json:"session_id" jsonschema:"description=Session ID to send input to,required"`
	Input     string `json:"input" jsonschema:"description=Input text to send,required"`
}

// registerSessionTools registers session-related MCP tools
func (s *ServerV2) registerSessionTools() error {
	// session_run tool
	runOpts, err := WithStructOptions("Run a task or command in a new session", SessionRunParams{})
	if err != nil {
		return fmt.Errorf("failed to create session_run options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("session_run", runOpts...), s.handleSessionRun)

	// session_list tool
	listOpts, err := WithStructOptions("List all sessions or sessions in a specific workspace", SessionListParams{})
	if err != nil {
		return fmt.Errorf("failed to create session_list options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("session_list", listOpts...), s.handleSessionList)

	// session_stop tool
	stopOpts, err := WithStructOptions("Stop a running session", SessionStopParams{})
	if err != nil {
		return fmt.Errorf("failed to create session_stop options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("session_stop", stopOpts...), s.handleSessionStop)

	// session_logs tool
	logsOpts, err := WithStructOptions("Get logs from a session", SessionLogsParams{})
	if err != nil {
		return fmt.Errorf("failed to create session_logs options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("session_logs", logsOpts...), s.handleSessionLogs)

	// session_remove tool
	removeOpts, err := WithStructOptions("Remove a stopped session", SessionRemoveParams{})
	if err != nil {
		return fmt.Errorf("failed to create session_remove options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("session_remove", removeOpts...), s.handleSessionRemove)

	// session_send_keys tool
	sendKeysOpts, err := WithStructOptions("Send input to a running session", SessionSendKeysParams{})
	if err != nil {
		return fmt.Errorf("failed to create session_send_keys options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("session_send_keys", sendKeysOpts...), s.handleSessionSendKeys)

	return nil
}

// handleSessionRun handles the session_run tool
func (s *ServerV2) handleSessionRun(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	// Parse parameters
	opts := session.CreateOptions{}

	if workspaceID, ok := args["workspace_id"].(string); ok {
		opts.WorkspaceID = workspaceID
	}
	// Enable auto-create by default in MCP unless explicitly disabled
	autoCreate := true
	if val, ok := args["auto_create_workspace"].(bool); ok {
		autoCreate = val
	}
	// Only enable auto-create if no workspace ID is provided
	if opts.WorkspaceID == "" {
		opts.AutoCreateWorkspace = autoCreate
	}
	if name, ok := args["name"].(string); ok {
		opts.Name = name
	}
	if description, ok := args["description"].(string); ok {
		opts.Description = description
	}
	if taskName, ok := args["task_name"].(string); ok {
		opts.TaskName = taskName
	}
	if cmdInterface, ok := args["command"].([]interface{}); ok {
		cmd := make([]string, len(cmdInterface))
		for i, v := range cmdInterface {
			if s, ok := v.(string); ok {
				cmd[i] = s
			}
		}
		opts.Command = cmd
	}
	if runtime, ok := args["runtime"].(string); ok {
		opts.Runtime = runtime
	}
	if env, ok := args["environment"].(map[string]interface{}); ok {
		envMap := make(map[string]string)
		for k, v := range env {
			if s, ok := v.(string); ok {
				envMap[k] = s
			}
		}
		opts.Environment = envMap
	}
	if workingDir, ok := args["working_dir"].(string); ok {
		opts.WorkingDir = workingDir
	}
	if enableLog, ok := args["enable_log"].(bool); ok {
		opts.EnableLog = enableLog
	}

	// Create session manager
	sessionMgr := s.getSessionManager()

	// Create session
	sess, err := sessionMgr.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	result := map[string]interface{}{
		"session_id":   sess.ID,
		"workspace_id": sess.WorkspaceID,
		"runtime":      sess.Runtime,
		"status":       sess.Status,
		"command":      sess.Command,
		"started_at":   sess.StartedAt,
		"message":      fmt.Sprintf("Session %s started successfully", sess.ID),
	}
	if sess.Name != "" {
		result["name"] = sess.Name
	}
	if sess.Description != "" {
		result["description"] = sess.Description
	}

	return createEnhancedResult("session_run", result, nil)
}

// handleSessionList handles the session_list tool
func (s *ServerV2) handleSessionList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	var workspaceID string
	if id, ok := args["workspace_id"].(string); ok {
		workspaceID = id
	}

	// Create session manager
	sessionMgr := s.getSessionManager()

	// List sessions
	sessions, err := sessionMgr.List(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Convert to response format
	result := make([]map[string]interface{}, len(sessions))
	for i, sess := range sessions {
		result[i] = map[string]interface{}{
			"session_id":   sess.ID,
			"workspace_id": sess.WorkspaceID,
			"task_name":    sess.TaskName,
			"runtime":      sess.Runtime,
			"status":       sess.Status,
			"command":      sess.Command,
			"started_at":   sess.StartedAt,
		}
		if sess.Name != "" {
			result[i]["name"] = sess.Name
		}
		if sess.Description != "" {
			result[i]["description"] = sess.Description
		}
		if sess.StoppedAt != nil {
			result[i]["stopped_at"] = sess.StoppedAt
		}
		if sess.ExitCode != nil {
			result[i]["exit_code"] = sess.ExitCode
		}
	}

	return createEnhancedResult("session_list", result, nil)
}

// handleSessionStop handles the session_stop tool
func (s *ServerV2) handleSessionStop(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	sessionID, ok := args["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing session_id argument")
	}

	force, _ := args["force"].(bool)

	// Create session manager
	sessionMgr := s.getSessionManager()

	// Stop or kill session
	if force {
		if err := sessionMgr.Kill(ctx, sessionID); err != nil {
			return nil, fmt.Errorf("failed to kill session: %w", err)
		}
		return createEnhancedResult("session_stop", map[string]interface{}{
			"message": fmt.Sprintf("Session %s killed", sessionID),
		}, nil)
	}

	if err := sessionMgr.Stop(ctx, sessionID); err != nil {
		return nil, fmt.Errorf("failed to stop session: %w", err)
	}

	return createEnhancedResult("session_stop", map[string]interface{}{
		"message": fmt.Sprintf("Session %s stopped", sessionID),
	}, nil)
}

// handleSessionLogs handles the session_logs tool
func (s *ServerV2) handleSessionLogs(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	sessionID, ok := args["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing session_id argument")
	}

	follow, _ := args["follow"].(bool)

	// Create session manager
	sessionMgr := s.getSessionManager()

	// Get logs
	reader, err := sessionMgr.Logs(ctx, sessionID, follow)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	// Read logs (limited for MCP response)
	buf := make([]byte, 64*1024) // 64KB limit
	n, _ := reader.Read(buf)

	return createEnhancedResult("session_logs", map[string]interface{}{
		"logs": string(buf[:n]),
		"note": "Logs are truncated to 64KB. Use CLI for full logs or streaming.",
	}, nil)
}

// handleSessionRemove handles the session_remove tool
func (s *ServerV2) handleSessionRemove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	sessionID, ok := args["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing session_id argument")
	}

	// Create session manager
	sessionMgr := s.getSessionManager()

	// Remove session
	if err := sessionMgr.Remove(ctx, sessionID); err != nil {
		return nil, fmt.Errorf("failed to remove session: %w", err)
	}

	return createEnhancedResult("session_remove", map[string]interface{}{
		"message": fmt.Sprintf("Session %s removed", sessionID),
	}, nil)
}

// handleSessionSendKeys handles the session_send_keys tool
func (s *ServerV2) handleSessionSendKeys(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	sessionID, ok := args["session_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing session_id argument")
	}

	input, ok := args["input"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing input argument")
	}

	// Create session manager
	sessionMgr := s.getSessionManager()

	// Send input to session
	if err := sessionMgr.SendInput(ctx, sessionID, input); err != nil {
		return nil, fmt.Errorf("failed to send input: %w", err)
	}

	return createEnhancedResult("session_send_keys", map[string]interface{}{
		"message": fmt.Sprintf("Input sent to session %s", sessionID),
	}, nil)
}

// getSessionManager creates a session manager for the server
func (s *ServerV2) getSessionManager() session.Manager {
	// Get runtimes
	runtimes := make(map[string]runtime.Runtime)
	for _, name := range runtime.List() {
		if rt, err := runtime.Get(name); err == nil {
			runtimes[name] = rt
		}
	}

	// Create task manager
	taskMgr := task.NewManager()
	// TODO: Load tasks from config

	// Create session store
	store := session.NewFileStore(s.configManager.GetAmuxDir())

	// Create session manager
	return session.NewManager(store, runtimes, taskMgr, s.workspaceManager, s.configManager)
}
