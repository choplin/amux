package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/aki/amux/internal/config"
	"github.com/aki/amux/internal/workspace"
)

// ServerV2 implements the MCP server using mcp-go
type ServerV2 struct {
	mcpServer  *server.MCPServer
	transport  string
	httpConfig *config.HTTPConfig

	// Manager references
	configManager    *config.Manager
	workspaceManager *workspace.Manager
}

// NewServerV2 creates a new MCP server using mcp-go
func NewServerV2(configManager *config.Manager, transport string, httpConfig *config.HTTPConfig) (*ServerV2, error) {
	// Create workspace manager
	workspaceManager, err := workspace.SetupManager(configManager.GetProjectRoot())
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
		mcpServer:        mcpServer,
		transport:        transport,
		httpConfig:       httpConfig,
		configManager:    configManager,
		workspaceManager: workspaceManager,
	}

	// Register all tools

	if err := s.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	// Register bridge tools
	if err := s.registerBridgeTools(); err != nil {
		return nil, fmt.Errorf("failed to register bridge tools: %w", err)
	}

	// Register session tools
	if err := s.registerSessionTools(); err != nil {
		return nil, fmt.Errorf("failed to register session tools: %w", err)
	}

	// Register all resources
	if err := s.registerResources(); err != nil {
		return nil, fmt.Errorf("failed to register resources: %w", err)
	}

	// Register resource templates
	if err := s.registerResourceTemplates(); err != nil {
		return nil, fmt.Errorf("failed to register resource templates: %w", err)
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

	createOpts, err := WithStructOptions(GetEnhancedDescription("workspace_create"), WorkspaceCreateParams{})
	if err != nil {
		return fmt.Errorf("failed to create workspace_create options: %w", err)
	}

	s.mcpServer.AddTool(mcp.NewTool("workspace_create", createOpts...), s.handleWorkspaceCreate)

	// workspace_remove tool

	removeOpts, err := WithStructOptions(GetEnhancedDescription("workspace_remove"), WorkspaceIDParams{})
	if err != nil {
		return fmt.Errorf("failed to create workspace_remove options: %w", err)
	}

	s.mcpServer.AddTool(mcp.NewTool("workspace_remove", removeOpts...), s.handleWorkspaceRemove)

	// Register storage tools
	if err := s.registerStorageTools(); err != nil {
		return fmt.Errorf("failed to register storage tools: %w", err)
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

	if description, ok := args["description"].(string); ok {
		opts.Description = description
	}

	ws, err := s.workspaceManager.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create enhanced result with metadata
	return createEnhancedResult("workspace_create", ws, nil)
}

func (s *ServerV2) handleWorkspaceRemove(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	workspaceID, ok := args["workspace_identifier"].(string)

	if !ok {
		return nil, fmt.Errorf("invalid or missing workspace_identifier argument")
	}

	// Resolve workspace to get name for better feedback
	ws, err := s.workspaceManager.ResolveWorkspace(ctx, workspace.Identifier(workspaceID))
	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			return nil, WorkspaceNotFoundError(workspaceID)
		}
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Get current working directory for safety check
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := s.workspaceManager.Remove(ctx, workspace.Identifier(ws.ID), workspace.RemoveOptions{
		NoHooks:    false,
		CurrentDir: cwd,
	}); err != nil {
		return nil, fmt.Errorf("failed to remove workspace: %w", err)
	}

	// Create enhanced result
	result := map[string]interface{}{
		"workspace_id":   ws.ID,
		"workspace_name": ws.Name,
		"message":        fmt.Sprintf("Workspace %s (%s) removed", ws.Name, ws.ID),
	}

	return createEnhancedResult("workspace_remove", result, nil)
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
