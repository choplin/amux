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
	mcpServer *server.MCPServer

	configManager *config.Manager

	workspaceManager *workspace.Manager

	transport string

	httpConfig *config.HTTPConfig
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

		mcpServer: mcpServer,

		configManager: configManager,

		workspaceManager: workspaceManager,

		transport: transport,

		httpConfig: httpConfig,
	}

	// Register all tools

	if err := s.registerTools(); err != nil {

		return nil, fmt.Errorf("failed to register tools: %w", err)

	}

	return s, nil

}

// registerTools registers all AgentCave tools

func (s *ServerV2) registerTools() error {

	// cave_create tool

	createOpts, err := WithStructOptions("Create a new isolated workspace", CaveCreateParams{})

	if err != nil {

		return fmt.Errorf("failed to create cave_create options: %w", err)

	}

	s.mcpServer.AddTool(mcp.NewTool("cave_create", createOpts...), s.handleCaveCreate)

	// cave_list tool

	listOpts, err := WithStructOptions("List all workspaces", CaveListParams{})

	if err != nil {

		return fmt.Errorf("failed to create cave_list options: %w", err)

	}

	s.mcpServer.AddTool(mcp.NewTool("cave_list", listOpts...), s.handleCaveList)

	// cave_get tool

	getOpts, err := WithStructOptions("Get workspace details", CaveIDParams{})

	if err != nil {

		return fmt.Errorf("failed to create cave_get options: %w", err)

	}

	s.mcpServer.AddTool(mcp.NewTool("cave_get", getOpts...), s.handleCaveGet)

	// cave_remove tool

	removeOpts, err := WithStructOptions("Remove workspace", CaveIDParams{})

	if err != nil {

		return fmt.Errorf("failed to create cave_remove options: %w", err)

	}

	s.mcpServer.AddTool(mcp.NewTool("cave_remove", removeOpts...), s.handleCaveRemove)

	// workspace_info tool

	workspaceInfoOpts, err := WithStructOptions("Browse workspace files and directories", WorkspaceInfoParams{})

	if err != nil {

		return fmt.Errorf("failed to create workspace_info options: %w", err)

	}

	s.mcpServer.AddTool(mcp.NewTool("workspace_info", workspaceInfoOpts...), s.handleWorkspaceInfo)

	return nil

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

	// No parameters needed, just list all workspaces

	workspaces, err := s.workspaceManager.List(workspace.ListOptions{})

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

	ws, err := s.workspaceManager.ResolveWorkspace(caveID)

	if err != nil {

		return nil, fmt.Errorf("failed to resolve workspace: %w", err)

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

func (s *ServerV2) handleCaveRemove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	args := request.GetArguments()

	caveID, ok := args["cave_id"].(string)

	if !ok {

		return nil, fmt.Errorf("invalid or missing cave_id argument")

	}

	// Resolve workspace to get name for better feedback

	ws, err := s.workspaceManager.ResolveWorkspace(caveID)

	if err != nil {

		return nil, fmt.Errorf("failed to resolve workspace: %w", err)

	}

	if err := s.workspaceManager.Remove(ws.ID); err != nil {

		return nil, fmt.Errorf("failed to remove workspace: %w", err)

	}

	return &mcp.CallToolResult{

		Content: []mcp.Content{

			mcp.TextContent{

				Type: "text",

				Text: fmt.Sprintf("Workspace %s (%s) removed", ws.Name, ws.ID),
			},
		},
	}, nil

}

func (s *ServerV2) handleWorkspaceInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	var params WorkspaceInfoParams

	if err := UnmarshalArgs(request, &params); err != nil {

		return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)

	}

	// Resolve workspace by name or ID

	ws, err := s.workspaceManager.ResolveWorkspace(params.CaveID)

	if err != nil {

		return nil, fmt.Errorf("failed to resolve workspace: %w", err)

	}

	// Validate path

	if err := git.ValidateWorktreePath(ws.Path, params.Path); err != nil {

		return nil, fmt.Errorf("invalid path: %w", err)

	}

	fullPath := filepath.Join(ws.Path, params.Path)

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

			"type": "directory",

			"path": params.Path,

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

		"type": "file",

		"path": params.Path,

		"size": info.Size(),

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

		Addr: fmt.Sprintf(":%d", s.httpConfig.Port),

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
