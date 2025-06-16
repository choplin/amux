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

// StorageReadParams represents parameters for reading storage files
type StorageReadParams struct {
	WorkspaceID string `json:"workspace_identifier" jsonschema:"title=Workspace Identifier,description=Workspace ID, index, or name"`
	SessionID   string `json:"session_identifier" jsonschema:"title=Session Identifier,description=Session ID, index, or name"`
	Path        string `json:"path" jsonschema:"title=Path,description=Relative path within storage directory"`
}

// StorageWriteParams represents parameters for writing storage files
type StorageWriteParams struct {
	WorkspaceID string `json:"workspace_identifier" jsonschema:"title=Workspace Identifier,description=Workspace ID, index, or name"`
	SessionID   string `json:"session_identifier" jsonschema:"title=Session Identifier,description=Session ID, index, or name"`
	Path        string `json:"path" jsonschema:"title=Path,description=Relative path within storage directory"`
	Content     string `json:"content" jsonschema:"title=Content,description=Content to write to the file"`
}

// StorageListParams represents parameters for listing storage contents
type StorageListParams struct {
	WorkspaceID string `json:"workspace_identifier" jsonschema:"title=Workspace Identifier,description=Workspace ID, index, or name"`
	SessionID   string `json:"session_identifier" jsonschema:"title=Session Identifier,description=Session ID, index, or name"`
	Path        string `json:"path,omitempty" jsonschema:"title=Path,description=Relative path within storage directory (optional)"`
}

// registerStorageTools registers storage-related tools
func (s *ServerV2) registerStorageTools() error {
	// Storage read tool
	readOpts, err := WithStructOptions(
		GetEnhancedDescription("storage_read"),
		StorageReadParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create storage_read options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("storage_read", readOpts...), s.handleStorageRead)

	// Storage write tool
	writeOpts, err := WithStructOptions(
		GetEnhancedDescription("storage_write"),
		StorageWriteParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create storage_write options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("storage_write", writeOpts...), s.handleStorageWrite)

	// Storage list tool
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

// handleStorageRead reads a file from storage
func (s *ServerV2) handleStorageRead(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	workspaceID, _ := args["workspace_identifier"].(string)
	sessionID, _ := args["session_identifier"].(string)
	path, _ := args["path"].(string)

	// Track whether we inferred parameters
	var metadata *ToolResultMetadata
	inferredWorkspace := false

	// Smart default: if no workspace or session specified, try to use the most recent workspace
	if workspaceID == "" && sessionID == "" {
		if recentWS, err := s.getMostRecentWorkspace(ctx); err == nil {
			workspaceID = recentWS.ID
			inferredWorkspace = true
		} else {
			return nil, fmt.Errorf("either workspace_identifier or session_identifier must be provided (no recent workspace found)")
		}
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

	// Set up metadata if we inferred parameters
	if inferredWorkspace {
		metadata = &ToolResultMetadata{
			InferredParameters: map[string]string{
				"workspace_identifier": workspaceID,
			},
		}
	}

	// Create result with file info
	result := map[string]interface{}{
		"path":    path,
		"content": string(content),
		"size":    len(content),
	}

	if inferredWorkspace {
		result["note"] = "Automatically used the most recent workspace"
	}

	return createEnhancedResult("storage_read", result, metadata)
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

	// Track if we inferred workspace
	var metadata *ToolResultMetadata
	if workspaceID != "" && sessionID == "" {
		// Check if this might be using the most recent workspace
		if recentWS, err := s.getMostRecentWorkspace(ctx); err == nil && recentWS.ID == workspaceID {
			metadata = &ToolResultMetadata{
				InferredParameters: map[string]string{
					"workspace_identifier": workspaceID,
				},
			}
		}
	}

	return createEnhancedResult("storage_write", result, metadata)
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

	// Track if we might have inferred workspace
	var metadata *ToolResultMetadata
	if workspaceID != "" && sessionID == "" {
		// Check if this might be using the most recent workspace
		if recentWS, err := s.getMostRecentWorkspace(ctx); err == nil && recentWS.ID == workspaceID {
			metadata = &ToolResultMetadata{
				InferredParameters: map[string]string{
					"workspace_identifier": workspaceID,
				},
			}
		}
	}

	return createEnhancedResult("storage_list", listResult, metadata)
}
