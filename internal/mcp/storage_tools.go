package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/storage"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/mark3labs/mcp-go/mcp"
)

// WorkspaceStorageReadParams represents parameters for reading workspace storage files
type WorkspaceStorageReadParams struct {
	WorkspaceID string `json:"workspace_identifier" jsonschema:"title=Workspace Identifier,description=Workspace ID, index, or name"`
	Path        string `json:"path" jsonschema:"title=Path,description=Relative path within storage directory"`
}

// WorkspaceStorageWriteParams represents parameters for writing workspace storage files
type WorkspaceStorageWriteParams struct {
	WorkspaceID string `json:"workspace_identifier" jsonschema:"title=Workspace Identifier,description=Workspace ID, index, or name"`
	Path        string `json:"path" jsonschema:"title=Path,description=Relative path within storage directory"`
	Content     string `json:"content" jsonschema:"title=Content,description=Content to write to the file"`
}

// WorkspaceStorageListParams represents parameters for listing workspace storage contents
type WorkspaceStorageListParams struct {
	WorkspaceID string `json:"workspace_identifier" jsonschema:"title=Workspace Identifier,description=Workspace ID, index, or name"`
	Path        string `json:"path,omitempty" jsonschema:"title=Path,description=Relative path within storage directory (optional)"`
}

// SessionStorageReadParams represents parameters for reading session storage files
type SessionStorageReadParams struct {
	SessionID string `json:"session_identifier" jsonschema:"title=Session Identifier,description=Session ID, index, or name"`
	Path      string `json:"path" jsonschema:"title=Path,description=Relative path within storage directory"`
}

// SessionStorageWriteParams represents parameters for writing session storage files
type SessionStorageWriteParams struct {
	SessionID string `json:"session_identifier" jsonschema:"title=Session Identifier,description=Session ID, index, or name"`
	Path      string `json:"path" jsonschema:"title=Path,description=Relative path within storage directory"`
	Content   string `json:"content" jsonschema:"title=Content,description=Content to write to the file"`
}

// SessionStorageListParams represents parameters for listing session storage contents
type SessionStorageListParams struct {
	SessionID string `json:"session_identifier" jsonschema:"title=Session Identifier,description=Session ID, index, or name"`
	Path      string `json:"path,omitempty" jsonschema:"title=Path,description=Relative path within storage directory (optional)"`
}

// registerStorageTools registers storage-related tools
func (s *ServerV2) registerStorageTools() error {
	// Register separated workspace storage tools
	if err := s.registerWorkspaceStorageTools(); err != nil {
		return err
	}

	// Register separated session storage tools
	if err := s.registerSessionStorageTools(); err != nil {
		return err
	}

	return nil
}

// registerWorkspaceStorageTools registers workspace-specific storage tools
func (s *ServerV2) registerWorkspaceStorageTools() error {
	// Workspace storage read
	readOpts, err := WithStructOptions(
		GetEnhancedDescription("workspace_storage_read"),
		WorkspaceStorageReadParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create workspace_storage_read options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("workspace_storage_read", readOpts...), s.handleWorkspaceStorageRead)

	// Workspace storage write
	writeOpts, err := WithStructOptions(
		GetEnhancedDescription("workspace_storage_write"),
		WorkspaceStorageWriteParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create workspace_storage_write options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("workspace_storage_write", writeOpts...), s.handleWorkspaceStorageWrite)

	// Workspace storage list
	listOpts, err := WithStructOptions(
		GetEnhancedDescription("workspace_storage_list"),
		WorkspaceStorageListParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create workspace_storage_list options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("workspace_storage_list", listOpts...), s.handleWorkspaceStorageList)

	return nil
}

// registerSessionStorageTools registers session-specific storage tools
func (s *ServerV2) registerSessionStorageTools() error {
	// Session storage read
	readOpts, err := WithStructOptions(
		GetEnhancedDescription("session_storage_read"),
		SessionStorageReadParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create session_storage_read options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("session_storage_read", readOpts...), s.handleSessionStorageRead)

	// Session storage write
	writeOpts, err := WithStructOptions(
		GetEnhancedDescription("session_storage_write"),
		SessionStorageWriteParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create session_storage_write options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("session_storage_write", writeOpts...), s.handleSessionStorageWrite)

	// Session storage list
	listOpts, err := WithStructOptions(
		GetEnhancedDescription("session_storage_list"),
		SessionStorageListParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create session_storage_list options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("session_storage_list", listOpts...), s.handleSessionStorageList)

	return nil
}

// Workspace storage handlers

// handleWorkspaceStorageRead reads a file from workspace storage
func (s *ServerV2) handleWorkspaceStorageRead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	workspaceID, _ := args["workspace_identifier"].(string)
	path, _ := args["path"].(string)

	// Get workspace storage path
	ws, err := s.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(workspaceID))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, WorkspaceNotFoundError(workspaceID)
		}
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Create storage manager
	storageManager := storage.NewManager(ws)

	// Read the file
	content, err := storageManager.ReadFile(ctx, path)
	if err != nil {
		if strings.Contains(err.Error(), "file not found") {
			return nil, FileNotFoundError(path)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Create result with file info
	result := map[string]interface{}{
		"path":    path,
		"content": string(content),
		"size":    len(content),
	}

	return createEnhancedResult("workspace_storage_read", result, nil)
}

// handleWorkspaceStorageWrite writes a file to workspace storage
func (s *ServerV2) handleWorkspaceStorageWrite(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	workspaceID, _ := args["workspace_identifier"].(string)
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	// Get workspace storage path
	ws, err := s.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(workspaceID))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, WorkspaceNotFoundError(workspaceID)
		}
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Create storage manager
	storageManager := storage.NewManager(ws)

	// Write the file
	if err := storageManager.WriteFile(ctx, path, []byte(content)); err != nil {
		return nil, err
	}

	// Create enhanced result
	result := map[string]interface{}{
		"path":    path,
		"bytes":   len(content),
		"message": fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path),
	}

	return createEnhancedResult("workspace_storage_write", result, nil)
}

// handleWorkspaceStorageList lists files in workspace storage
func (s *ServerV2) handleWorkspaceStorageList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	workspaceID, _ := args["workspace_identifier"].(string)
	subPath, _ := args["path"].(string)

	// Get workspace storage path
	ws, err := s.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(workspaceID))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, WorkspaceNotFoundError(workspaceID)
		}
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Create storage manager
	storageManager := storage.NewManager(ws)

	// List files
	listResult, err := storageManager.ListFiles(ctx, subPath)
	if err != nil {
		if strings.Contains(err.Error(), "path does not exist") {
			return nil, DirectoryNotFoundError(subPath)
		}
		return nil, err
	}

	var files []string
	for _, file := range listResult.Files {
		name := file.Name
		if file.IsDir {
			name += "/"
		}
		files = append(files, name)
	}

	// Create enhanced result
	result := map[string]interface{}{
		"path":  subPath,
		"files": files,
		"count": len(files),
	}

	return createEnhancedResult("workspace_storage_list", result, nil)
}

// Session storage handlers

// handleSessionStorageRead reads a file from session storage
func (s *ServerV2) handleSessionStorageRead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	sessionID, _ := args["session_identifier"].(string)
	path, _ := args["path"].(string)

	// Get session storage path
	sessionManager, err := s.createSessionManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}
	sess, err := sessionManager.ResolveSession(ctx, session.Identifier(sessionID))
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	// Create storage manager
	storageManager := storage.NewManager(sess)

	// Read the file
	content, err := storageManager.ReadFile(ctx, path)
	if err != nil {
		if strings.Contains(err.Error(), "file not found") {
			return nil, FileNotFoundError(path)
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Create result with file info
	result := map[string]interface{}{
		"path":    path,
		"content": string(content),
		"size":    len(content),
	}

	return createEnhancedResult("session_storage_read", result, nil)
}

// handleSessionStorageWrite writes a file to session storage
func (s *ServerV2) handleSessionStorageWrite(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	sessionID, _ := args["session_identifier"].(string)
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	// Get session storage path
	sessionManager, err := s.createSessionManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}
	sess, err := sessionManager.ResolveSession(ctx, session.Identifier(sessionID))
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	// Create storage manager
	storageManager := storage.NewManager(sess)

	// Write the file
	if err := storageManager.WriteFile(ctx, path, []byte(content)); err != nil {
		return nil, err
	}

	// Create enhanced result
	result := map[string]interface{}{
		"path":    path,
		"bytes":   len(content),
		"message": fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path),
	}

	return createEnhancedResult("session_storage_write", result, nil)
}

// handleSessionStorageList lists files in session storage
func (s *ServerV2) handleSessionStorageList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	sessionID, _ := args["session_identifier"].(string)
	subPath, _ := args["path"].(string)

	// Get session storage path
	sessionManager, err := s.createSessionManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}
	sess, err := sessionManager.ResolveSession(ctx, session.Identifier(sessionID))
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	// Create storage manager
	storageManager := storage.NewManager(sess)

	// List files
	listResult, err := storageManager.ListFiles(ctx, subPath)
	if err != nil {
		if strings.Contains(err.Error(), "path does not exist") {
			return nil, DirectoryNotFoundError(subPath)
		}
		return nil, err
	}

	var files []string
	for _, file := range listResult.Files {
		name := file.Name
		if file.IsDir {
			name += "/"
		}
		files = append(files, name)
	}

	// Create enhanced result
	result := map[string]interface{}{
		"path":  subPath,
		"files": files,
		"count": len(files),
	}

	return createEnhancedResult("session_storage_list", result, nil)
}
