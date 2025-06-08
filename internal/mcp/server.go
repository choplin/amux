package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aki/agentcave/internal/core/config"
	"github.com/aki/agentcave/internal/core/git"
	"github.com/aki/agentcave/internal/core/workspace"
)

// ServerV2 implements the MCP server using mcp-go
type ServerV2 struct {
	mcpServer        *server.MCPServer
	configManager    *config.Manager
	workspaceManager *workspace.Manager
	transport        string
	httpConfig       *config.HTTPConfig
}

// NewServerV2 creates a new MCP server using mcp-go
func NewServerV2(configManager *config.Manager, transport string, httpConfig *config.HTTPConfig) (*ServerV2, error) {
	workspaceManager, err := workspace.NewManager(configManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"agentcave",
		"1.0.0",
		server.WithLogging(),
	)

	s := &ServerV2{
		mcpServer:        mcpServer,
		configManager:    configManager,
		workspaceManager: workspaceManager,
		transport:        transport,
		httpConfig:       httpConfig,
	}

	// Register all tools
	s.registerTools()

	return s, nil
}

// registerTools registers all AgentCave tools
func (s *ServerV2) registerTools() {
	// cave_create tool
	s.mcpServer.AddTool(mcp.NewTool("cave_create",
		mcp.WithDescription("Create a new isolated workspace"),
		mcp.WithString("name",
			mcp.Description("Workspace name"),
			mcp.Required(),
		),
		mcp.WithString("baseBranch",
			mcp.Description("Base branch (optional)"),
		),
		mcp.WithString("branch",
			mcp.Description("Use existing branch (optional)"),
		),
		mcp.WithString("agentId",
			mcp.Description("Agent ID (optional)"),
		),
		mcp.WithString("description",
			mcp.Description("Description (optional)"),
		),
	), s.handleCaveCreate)

	// cave_list tool
	s.mcpServer.AddTool(mcp.NewTool("cave_list",
		mcp.WithDescription("List all workspaces"),
		mcp.WithString("status",
			mcp.Description("Filter by status (optional)"),
			mcp.Enum("active", "idle"),
		),
	), s.handleCaveList)

	// cave_get tool
	s.mcpServer.AddTool(mcp.NewTool("cave_get",
		mcp.WithDescription("Get workspace details"),
		mcp.WithString("cave_id",
			mcp.Description("Workspace ID"),
			mcp.Required(),
		),
	), s.handleCaveGet)

	// cave_activate tool
	s.mcpServer.AddTool(mcp.NewTool("cave_activate",
		mcp.WithDescription("Mark workspace as active"),
		mcp.WithString("cave_id",
			mcp.Description("Workspace ID"),
			mcp.Required(),
		),
	), s.handleCaveActivate)

	// cave_deactivate tool
	s.mcpServer.AddTool(mcp.NewTool("cave_deactivate",
		mcp.WithDescription("Mark workspace as idle"),
		mcp.WithString("cave_id",
			mcp.Description("Workspace ID"),
			mcp.Required(),
		),
	), s.handleCaveDeactivate)

	// cave_remove tool
	s.mcpServer.AddTool(mcp.NewTool("cave_remove",
		mcp.WithDescription("Remove workspace"),
		mcp.WithString("cave_id",
			mcp.Description("Workspace ID"),
			mcp.Required(),
		),
	), s.handleCaveRemove)

	// workspace_info tool
	s.mcpServer.AddTool(mcp.NewTool("workspace_info",
		mcp.WithDescription("Browse workspace files and directories"),
		mcp.WithString("cave_id",
			mcp.Description("Workspace ID"),
			mcp.Required(),
		),
		mcp.WithString("path",
			mcp.Description("File or directory path (optional)"),
		),
	), s.handleWorkspaceInfo)
}

// Tool handlers

func (s *ServerV2) handleCaveCreate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	name, ok := args["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing name argument")
	}

	opts := workspace.CreateOptions{
		Name: name,
	}

	// Optional parameters
	if baseBranch, ok := args["baseBranch"].(string); ok {
		opts.BaseBranch = baseBranch
	}
	if branch, ok := args["branch"].(string); ok {
		opts.Branch = branch
	}
	if agentID, ok := args["agentId"].(string); ok {
		opts.AgentID = agentID
	}
	if description, ok := args["description"].(string); ok {
		opts.Description = description
	}

	ws, err := s.workspaceManager.Create(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	result, _ := json.MarshalIndent(ws, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(result),
			},
		},
	}, nil
}

func (s *ServerV2) handleCaveList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	opts := workspace.ListOptions{}
	if status, ok := args["status"].(string); ok {
		opts.Status = workspace.Status(status)
	}

	workspaces, err := s.workspaceManager.List(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	result, _ := json.MarshalIndent(workspaces, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(result),
			},
		},
	}, nil
}

func (s *ServerV2) handleCaveGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	caveID, ok := args["cave_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing cave_id argument")
	}

	ws, err := s.workspaceManager.Get(caveID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	result, _ := json.MarshalIndent(ws, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(result),
			},
		},
	}, nil
}

func (s *ServerV2) handleCaveActivate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	caveID, ok := args["cave_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing cave_id argument")
	}

	if err := s.workspaceManager.Activate(caveID); err != nil {
		return nil, fmt.Errorf("failed to activate workspace: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Workspace %s activated", caveID),
			},
		},
	}, nil
}

func (s *ServerV2) handleCaveDeactivate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	caveID, ok := args["cave_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing cave_id argument")
	}

	if err := s.workspaceManager.Deactivate(caveID); err != nil {
		return nil, fmt.Errorf("failed to deactivate workspace: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Workspace %s deactivated", caveID),
			},
		},
	}, nil
}

func (s *ServerV2) handleCaveRemove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	caveID, ok := args["cave_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing cave_id argument")
	}

	if err := s.workspaceManager.Remove(caveID); err != nil {
		return nil, fmt.Errorf("failed to remove workspace: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Workspace %s removed", caveID),
			},
		},
	}, nil
}

func (s *ServerV2) handleWorkspaceInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	caveID, ok := args["cave_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing cave_id argument")
	}

	// Get workspace
	ws, err := s.workspaceManager.Get(caveID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	// Get optional path
	path := ""
	if p, ok := args["path"].(string); ok {
		path = p
	}

	// Validate path
	if err := git.ValidateWorktreePath(ws.Path, path); err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	fullPath := filepath.Join(ws.Path, path)

	// Check if path exists
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("path not found: %w", err)
	}

	if info.IsDir() {
		// List directory contents
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read directory: %w", err)
		}

		var files []map[string]interface{}
		for _, entry := range entries {
			fileInfo := map[string]interface{}{
				"name": entry.Name(),
				"type": "file",
				"size": 0,
			}

			if entry.IsDir() {
				fileInfo["type"] = "directory"
			} else {
				if info, err := entry.Info(); err == nil {
					fileInfo["size"] = info.Size()
				}
			}

			files = append(files, fileInfo)
		}

		result, _ := json.MarshalIndent(map[string]interface{}{
			"type":  "directory",
			"path":  path,
			"files": files,
		}, "", "  ")

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: string(result),
				},
			},
		}, nil
	}

	// Read file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Limit file size
	const maxFileSize = 1024 * 1024 // 1MB
	if len(content) > maxFileSize {
		content = content[:maxFileSize]
		content = append(content, []byte("\n\n[File truncated - exceeds 1MB limit]")...)
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"type":    "file",
		"path":    path,
		"size":    info.Size(),
		"content": string(content),
	}, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(result),
			},
		},
	}, nil
}

// Start starts the MCP server
func (s *ServerV2) Start(ctx context.Context) error {
	switch s.transport {
	case "stdio":
		return server.ServeStdio(s.mcpServer)
	case "https", "http":
		return s.startHTTPServer(ctx)
	default:
		return fmt.Errorf("unsupported transport: %s", s.transport)
	}
}

// startHTTPServer starts the HTTP/SSE server
func (s *ServerV2) startHTTPServer(ctx context.Context) error {
	if s.httpConfig == nil {
		return fmt.Errorf("HTTP configuration required")
	}

	// Create SSE server
	sseServer := server.NewSSEServer(s.mcpServer)

	// Create HTTP mux
	mux := http.NewServeMux()

	// Add SSE endpoints
	mux.Handle("/sse", sseServer.SSEHandler())
	mux.Handle("/message", sseServer.MessageHandler())

	// Apply middleware
	handler := s.corsMiddleware(s.authMiddleware(mux))

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.httpConfig.Port),
		Handler: handler,
	}

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to shutdown server: %v\n", err)
		}
	}()

	fmt.Printf("MCP server listening on http://localhost:%d\n", s.httpConfig.Port)
	fmt.Printf("SSE endpoint: http://localhost:%d/sse\n", s.httpConfig.Port)
	fmt.Printf("Message endpoint: http://localhost:%d/message\n", s.httpConfig.Port)

	return httpServer.ListenAndServe()
}

// corsMiddleware adds CORS headers
func (s *ServerV2) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// authMiddleware handles authentication
func (s *ServerV2) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.httpConfig.Auth.Type == "" || s.httpConfig.Auth.Type == "none" {
			next.ServeHTTP(w, r)
			return
		}

		switch s.httpConfig.Auth.Type {
		case "bearer":
			token := r.Header.Get("Authorization")
			expectedToken := fmt.Sprintf("Bearer %s", s.httpConfig.Auth.Bearer)
			if token != expectedToken {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

		case "basic":
			username, password, ok := r.BasicAuth()
			if !ok || username != s.httpConfig.Auth.Basic.Username || password != s.httpConfig.Auth.Basic.Password {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

		default:
			http.Error(w, "Invalid auth type", http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r)
	})
}
