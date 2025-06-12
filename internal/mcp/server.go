package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/mark3labs/mcp-go/server"

	"github.com/aki/amux/internal/core/config"

	"github.com/aki/amux/internal/core/workspace"
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

		"amux",

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

	// Register bridge tools
	if err := s.registerBridgeTools(); err != nil {
		return nil, fmt.Errorf("failed to register bridge tools: %w", err)
	}

	// Register all resources
	if err := s.registerResources(); err != nil {
		return nil, fmt.Errorf("failed to register resources: %w", err)
	}

	// Register resource templates
	if err := s.registerResourceTemplates(); err != nil {
		return nil, fmt.Errorf("failed to register resource templates: %w", err)
	}

	// Register session resources
	if err := s.registerSessionResources(); err != nil {
		return nil, fmt.Errorf("failed to register session resources: %w", err)
	}

	// Register all prompts
	if err := s.registerPrompts(); err != nil {
		return nil, fmt.Errorf("failed to register prompts: %w", err)
	}

	return s, nil
}

// registerTools registers all Amux tools

func (s *ServerV2) registerTools() error {
	// workspace_create tool

	createOpts, err := WithStructOptions("Create a new isolated git worktree-based workspace for development. Each workspace has its own branch and can be used for working on separate features or issues", WorkspaceCreateParams{})
	if err != nil {
		return fmt.Errorf("failed to create workspace_create options: %w", err)
	}

	s.mcpServer.AddTool(mcp.NewTool("workspace_create", createOpts...), s.handleWorkspaceCreate)

	// workspace_remove tool

	removeOpts, err := WithStructOptions("Remove a workspace and its associated git worktree. This permanently deletes the workspace directory and cannot be undone", WorkspaceIDParams{})
	if err != nil {
		return fmt.Errorf("failed to create workspace_remove options: %w", err)
	}

	s.mcpServer.AddTool(mcp.NewTool("workspace_remove", removeOpts...), s.handleWorkspaceRemove)

	// Register session tools
	if err := s.registerSessionTools(); err != nil {
		return fmt.Errorf("failed to register session tools: %w", err)
	}

	return nil
}

// Tool handlers

func (s *ServerV2) handleWorkspaceCreate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func (s *ServerV2) handleWorkspaceRemove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	workspaceID, ok := args["workspace_id"].(string)

	if !ok {
		return nil, fmt.Errorf("invalid or missing workspace_id argument")
	}

	// Resolve workspace to get name for better feedback

	ws, err := s.workspaceManager.ResolveWorkspace(workspaceID)
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

// Start starts the MCP server
func (s *ServerV2) Start(ctx context.Context) error {
	switch s.transport {

	case "stdio":

		// Create stdio server with custom error logging
		stdioServer := server.NewStdioServer(s.mcpServer)

		// Log to stderr to avoid interfering with stdio protocol
		logger := log.New(os.Stderr, "[AMUX MCP] ", log.LstdFlags|log.Lshortfile)
		stdioServer.SetErrorLogger(logger)

		// Don't use ServeStdio as it sets up its own signal handling
		// Use Listen directly with the provided context
		return stdioServer.Listen(ctx, os.Stdin, os.Stdout)

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

		// Create new context for shutdown since parent is already cancelled
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil { //nolint:contextcheck

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
