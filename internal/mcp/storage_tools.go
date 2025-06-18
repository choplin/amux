package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/mark3labs/mcp-go/mcp"
)

// StorageReadParams represents parameters for reading storage files (deprecated)
type StorageReadParams struct {
	WorkspaceID string `json:"workspace_identifier" jsonschema:"title=Workspace Identifier,description=Workspace ID, index, or name"`
	SessionID   string `json:"session_identifier" jsonschema:"title=Session Identifier,description=Session ID, index, or name"`
	Path        string `json:"path" jsonschema:"title=Path,description=Relative path within storage directory"`
}

// StorageWriteParams represents parameters for writing storage files (deprecated)
type StorageWriteParams struct {
	WorkspaceID string `json:"workspace_identifier" jsonschema:"title=Workspace Identifier,description=Workspace ID, index, or name"`
	SessionID   string `json:"session_identifier" jsonschema:"title=Session Identifier,description=Session ID, index, or name"`
	Path        string `json:"path" jsonschema:"title=Path,description=Relative path within storage directory"`
	Content     string `json:"content" jsonschema:"title=Content,description=Content to write to the file"`
}

// StorageListParams represents parameters for listing storage contents (deprecated)
type StorageListParams struct {
	WorkspaceID string `json:"workspace_identifier" jsonschema:"title=Workspace Identifier,description=Workspace ID, index, or name"`
	SessionID   string `json:"session_identifier" jsonschema:"title=Session Identifier,description=Session ID, index, or name"`
	Path        string `json:"path,omitempty" jsonschema:"title=Path,description=Relative path within storage directory (optional)"`
}

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
	// Register deprecated unified tools
	if err := s.registerDeprecatedStorageTools(); err != nil {
		return err
	}

	// Register new separated workspace storage tools
	if err := s.registerWorkspaceStorageTools(); err != nil {
		return err
	}

	// Register new separated session storage tools
	if err := s.registerSessionStorageTools(); err != nil {
		return err
	}

	return nil
}

// registerDeprecatedStorageTools registers the deprecated unified storage tools
func (s *ServerV2) registerDeprecatedStorageTools() error {
	// Storage read tool (deprecated)
	readOpts, err := WithStructOptions(
		GetEnhancedDescription("storage_read"),
		StorageReadParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create storage_read options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("storage_read", readOpts...), s.handleStorageRead)

	// Storage write tool (deprecated)
	writeOpts, err := WithStructOptions(
		GetEnhancedDescription("storage_write"),
		StorageWriteParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create storage_write options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("storage_write", writeOpts...), s.handleStorageWrite)

	// Storage list tool (deprecated)
	listOpts, err := WithStructOptions(
		GetEnhancedDescription("storage_list"),
		StorageListParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create storage_list options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("storage_list", listOpts...), s.handleStorageList)

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

// handleStorageRead reads a file from storage
func (s *ServerV2) handleStorageRead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	workspaceID, _ := args["workspace_identifier"].(string)
	sessionID, _ := args["session_identifier"].(string)
	path, _ := args["path"].(string)

	if workspaceID == "" && sessionID == "" {
		return nil, fmt.Errorf("either workspace_identifier or session_identifier must be provided")
	}

	if workspaceID != "" && sessionID != "" {
		return nil, fmt.Errorf("only one of workspace_identifier or session_identifier should be provided")
	}

	var storagePath string
	if workspaceID != "" {
		// Get workspace storage path
		ws, err := s.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(workspaceID))
		if err != nil {
			// Check if it's a not found error
			if strings.Contains(err.Error(), "not found") {
				return nil, WorkspaceNotFoundError(workspaceID)
			}
			return nil, fmt.Errorf("failed to resolve workspace: %w", err)
		}
		storagePath = ws.StoragePath
	} else {
		// Get session storage path
		sessionManager, err := s.createSessionManager()
		if err != nil {
			return nil, fmt.Errorf("failed to create session manager: %w", err)
		}
		sess, err := sessionManager.ResolveSession(ctx, session.Identifier(sessionID))
		if err != nil {
			return nil, fmt.Errorf("failed to get session: %w", err)
		}
		storagePath = sess.Info().StoragePath
	}

	if storagePath == "" {
		return nil, fmt.Errorf("storage path not found")
	}

	// Construct full path
	fullPath := filepath.Join(storagePath, path)

	// Ensure the path is within the storage directory
	cleanPath := filepath.Clean(fullPath)
	cleanStoragePath := filepath.Clean(storagePath)
	if !strings.HasPrefix(cleanPath, cleanStoragePath) {
		return nil, fmt.Errorf("path traversal attempt detected")
	}

	// Read the file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		// Check if it's a file not found error
		if os.IsNotExist(err) {
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

	return createEnhancedResult("storage_read", result, nil)
}

// handleStorageWrite writes a file to storage
func (s *ServerV2) handleStorageWrite(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	workspaceID, _ := args["workspace_identifier"].(string)
	sessionID, _ := args["session_identifier"].(string)
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	if workspaceID == "" && sessionID == "" {
		return nil, fmt.Errorf("either workspace_identifier or session_identifier must be provided")
	}

	if workspaceID != "" && sessionID != "" {
		return nil, fmt.Errorf("only one of workspace_identifier or session_identifier should be provided")
	}

	var storagePath string
	if workspaceID != "" {
		// Get workspace storage path
		ws, err := s.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(workspaceID))
		if err != nil {
			// Check if it's a not found error
			if strings.Contains(err.Error(), "not found") {
				return nil, WorkspaceNotFoundError(workspaceID)
			}
			return nil, fmt.Errorf("failed to resolve workspace: %w", err)
		}
		storagePath = ws.StoragePath
	} else {
		// Get session storage path
		sessionManager, err := s.createSessionManager()
		if err != nil {
			return nil, fmt.Errorf("failed to create session manager: %w", err)
		}
		sess, err := sessionManager.ResolveSession(ctx, session.Identifier(sessionID))
		if err != nil {
			return nil, fmt.Errorf("failed to get session: %w", err)
		}
		storagePath = sess.Info().StoragePath
	}

	if storagePath == "" {
		return nil, fmt.Errorf("storage path not found")
	}

	// Construct full path
	fullPath := filepath.Join(storagePath, path)

	// Ensure the path is within the storage directory
	cleanPath := filepath.Clean(fullPath)
	cleanStoragePath := filepath.Clean(storagePath)
	if !strings.HasPrefix(cleanPath, cleanStoragePath) {
		return nil, fmt.Errorf("path traversal attempt detected")
	}

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// Create enhanced result
	result := map[string]interface{}{
		"path":    path,
		"bytes":   len(content),
		"message": fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path),
	}

	return createEnhancedResult("storage_write", result, nil)
}

// handleStorageList lists files in storage
func (s *ServerV2) handleStorageList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	workspaceID, _ := args["workspace_identifier"].(string)
	sessionID, _ := args["session_identifier"].(string)
	subPath, _ := args["path"].(string)

	if workspaceID == "" && sessionID == "" {
		return nil, fmt.Errorf("either workspace_identifier or session_identifier must be provided")
	}

	if workspaceID != "" && sessionID != "" {
		return nil, fmt.Errorf("only one of workspace_identifier or session_identifier should be provided")
	}

	var storagePath string
	if workspaceID != "" {
		// Get workspace storage path
		ws, err := s.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(workspaceID))
		if err != nil {
			// Check if it's a not found error
			if strings.Contains(err.Error(), "not found") {
				return nil, WorkspaceNotFoundError(workspaceID)
			}
			return nil, fmt.Errorf("failed to resolve workspace: %w", err)
		}
		storagePath = ws.StoragePath
	} else {
		// Get session storage path
		sessionManager, err := s.createSessionManager()
		if err != nil {
			return nil, fmt.Errorf("failed to create session manager: %w", err)
		}
		sess, err := sessionManager.ResolveSession(ctx, session.Identifier(sessionID))
		if err != nil {
			return nil, fmt.Errorf("failed to get session: %w", err)
		}
		storagePath = sess.Info().StoragePath
	}

	if storagePath == "" {
		return nil, fmt.Errorf("storage path not found")
	}

	// Construct full path
	listPath := storagePath
	if subPath != "" {
		listPath = filepath.Join(storagePath, subPath)
		// Ensure the path is within the storage directory
		cleanListPath := filepath.Clean(listPath)
		cleanStoragePath := filepath.Clean(storagePath)
		if !strings.HasPrefix(cleanListPath, cleanStoragePath) {
			return nil, fmt.Errorf("path traversal attempt detected")
		}
	}

	// List directory contents
	entries, err := os.ReadDir(listPath)
	if err != nil {
		// Check if it's a directory not found error
		if os.IsNotExist(err) {
			return nil, DirectoryNotFoundError(subPath)
		}
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		files = append(files, name)
	}

	// Just remove the old formatting code - we're using structured data now

	// Create enhanced result
	listResult := map[string]interface{}{
		"path":  subPath,
		"files": files,
		"count": len(files),
	}

	return createEnhancedResult("storage_list", listResult, nil)
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
	storagePath := ws.StoragePath

	if storagePath == "" {
		return nil, fmt.Errorf("storage path not found")
	}

	// Construct full path
	fullPath := filepath.Join(storagePath, path)

	// Ensure the path is within the storage directory
	cleanPath := filepath.Clean(fullPath)
	cleanStoragePath := filepath.Clean(storagePath)
	if !strings.HasPrefix(cleanPath, cleanStoragePath) {
		return nil, fmt.Errorf("path traversal attempt detected")
	}

	// Read the file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
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
	storagePath := ws.StoragePath

	if storagePath == "" {
		return nil, fmt.Errorf("storage path not found")
	}

	// Construct full path
	fullPath := filepath.Join(storagePath, path)

	// Ensure the path is within the storage directory
	cleanPath := filepath.Clean(fullPath)
	cleanStoragePath := filepath.Clean(storagePath)
	if !strings.HasPrefix(cleanPath, cleanStoragePath) {
		return nil, fmt.Errorf("path traversal attempt detected")
	}

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
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
	storagePath := ws.StoragePath

	if storagePath == "" {
		return nil, fmt.Errorf("storage path not found")
	}

	// Construct full path
	listPath := storagePath
	if subPath != "" {
		listPath = filepath.Join(storagePath, subPath)
		// Ensure the path is within the storage directory
		cleanListPath := filepath.Clean(listPath)
		cleanStoragePath := filepath.Clean(storagePath)
		if !strings.HasPrefix(cleanListPath, cleanStoragePath) {
			return nil, fmt.Errorf("path traversal attempt detected")
		}
	}

	// List directory contents
	entries, err := os.ReadDir(listPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, DirectoryNotFoundError(subPath)
		}
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		files = append(files, name)
	}

	// Create enhanced result
	listResult := map[string]interface{}{
		"path":  subPath,
		"files": files,
		"count": len(files),
	}

	return createEnhancedResult("workspace_storage_list", listResult, nil)
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
	storagePath := sess.Info().StoragePath

	if storagePath == "" {
		return nil, fmt.Errorf("storage path not found")
	}

	// Construct full path
	fullPath := filepath.Join(storagePath, path)

	// Ensure the path is within the storage directory
	cleanPath := filepath.Clean(fullPath)
	cleanStoragePath := filepath.Clean(storagePath)
	if !strings.HasPrefix(cleanPath, cleanStoragePath) {
		return nil, fmt.Errorf("path traversal attempt detected")
	}

	// Read the file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
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
	storagePath := sess.Info().StoragePath

	if storagePath == "" {
		return nil, fmt.Errorf("storage path not found")
	}

	// Construct full path
	fullPath := filepath.Join(storagePath, path)

	// Ensure the path is within the storage directory
	cleanPath := filepath.Clean(fullPath)
	cleanStoragePath := filepath.Clean(storagePath)
	if !strings.HasPrefix(cleanPath, cleanStoragePath) {
		return nil, fmt.Errorf("path traversal attempt detected")
	}

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
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
	storagePath := sess.Info().StoragePath

	if storagePath == "" {
		return nil, fmt.Errorf("storage path not found")
	}

	// Construct full path
	listPath := storagePath
	if subPath != "" {
		listPath = filepath.Join(storagePath, subPath)
		// Ensure the path is within the storage directory
		cleanListPath := filepath.Clean(listPath)
		cleanStoragePath := filepath.Clean(storagePath)
		if !strings.HasPrefix(cleanListPath, cleanStoragePath) {
			return nil, fmt.Errorf("path traversal attempt detected")
		}
	}

	// List directory contents
	entries, err := os.ReadDir(listPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, DirectoryNotFoundError(subPath)
		}
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		files = append(files, name)
	}

	// Create enhanced result
	listResult := map[string]interface{}{
		"path":  subPath,
		"files": files,
		"count": len(files),
	}

	return createEnhancedResult("session_storage_list", listResult, nil)
}
