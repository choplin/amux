package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aki/agentcave/internal/core/config"
	"github.com/aki/agentcave/internal/core/git"
	"github.com/aki/agentcave/internal/core/workspace"
)

// Server implements the MCP server
type Server struct {
	configManager    *config.Manager
	workspaceManager *workspace.Manager
	transport        string
	httpConfig       *config.HTTPConfig
}

// NewServer creates a new MCP server
func NewServer(configManager *config.Manager, transport string, httpConfig *config.HTTPConfig) (*Server, error) {
	workspaceManager, err := workspace.NewManager(configManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	return &Server{
		configManager:    configManager,
		workspaceManager: workspaceManager,
		transport:        transport,
		httpConfig:       httpConfig,
	}, nil
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	switch s.transport {
	case "stdio":
		return s.startStdio(ctx)
	case "https", "http":
		return s.startHTTP(ctx)
	default:
		return fmt.Errorf("unsupported transport: %s", s.transport)
	}
}

// startStdio starts the stdio transport
func (s *Server) startStdio(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read line
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			// Parse request
			var req Request
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				resp := NewErrorResponse(nil, ParseError, "Parse error", err.Error())
				s.writeResponse(writer, resp)
				continue
			}

			// Handle request
			resp := s.handleRequest(&req)
			if err := s.writeResponse(writer, resp); err != nil {
				return err
			}
		}
	}
}

// startHTTP starts the HTTP transport
func (s *Server) startHTTP(ctx context.Context) error {
	if s.httpConfig == nil {
		return fmt.Errorf("HTTP configuration required")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHTTP)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.httpConfig.Port),
		Handler: s.corsMiddleware(s.authMiddleware(mux)),
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	fmt.Printf("MCP server listening on http://localhost:%d\n", s.httpConfig.Port)
	return server.ListenAndServe()
}

// handleHTTP handles HTTP requests
func (s *Server) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := NewErrorResponse(nil, ParseError, "Parse error", err.Error())
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := s.handleRequest(&req)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// authMiddleware handles authentication
func (s *Server) authMiddleware(next http.Handler) http.Handler {
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

// handleRequest processes a JSON-RPC request
func (s *Server) handleRequest(req *Request) *Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleListTools(req)
	case "tools/call":
		return s.handleCallTool(req)
	default:
		return NewErrorResponse(req.ID, MethodNotFound, "Method not found", nil)
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req *Request) *Response {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: ServerCapabilities{
			Tools: ToolsCapability{
				ListChanged: false,
			},
		},
		ServerInfo: ServerInfo{
			Name:    "agentcave",
			Version: "1.0.0",
		},
	}

	return NewResponse(req.ID, result)
}

// handleListTools handles the tools/list request
func (s *Server) handleListTools(req *Request) *Response {
	tools := []Tool{
		{
			Name:        "cave_create",
			Description: "Create a new isolated workspace",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {"type": "string", "description": "Workspace name"},
					"baseBranch": {"type": "string", "description": "Base branch (optional)"},
					"agentId": {"type": "string", "description": "Agent ID (optional)"},
					"description": {"type": "string", "description": "Description (optional)"}
				},
				"required": ["name"]
			}`),
		},
		{
			Name:        "cave_list",
			Description: "List all workspaces",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"status": {"type": "string", "enum": ["active", "idle"], "description": "Filter by status (optional)"}
				}
			}`),
		},
		{
			Name:        "cave_get",
			Description: "Get workspace details",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"cave_id": {"type": "string", "description": "Workspace ID"}
				},
				"required": ["cave_id"]
			}`),
		},
		{
			Name:        "cave_activate",
			Description: "Mark workspace as active",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"cave_id": {"type": "string", "description": "Workspace ID"}
				},
				"required": ["cave_id"]
			}`),
		},
		{
			Name:        "cave_deactivate",
			Description: "Mark workspace as idle",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"cave_id": {"type": "string", "description": "Workspace ID"}
				},
				"required": ["cave_id"]
			}`),
		},
		{
			Name:        "cave_remove",
			Description: "Remove workspace",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"cave_id": {"type": "string", "description": "Workspace ID"}
				},
				"required": ["cave_id"]
			}`),
		},
		{
			Name:        "workspace_info",
			Description: "Browse workspace files and directories",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"cave_id": {"type": "string", "description": "Workspace ID"},
					"path": {"type": "string", "description": "File or directory path (optional)"}
				},
				"required": ["cave_id"]
			}`),
		},
	}

	return NewResponse(req.ID, ListToolsResult{Tools: tools})
}

// handleCallTool handles the tools/call request
func (s *Server) handleCallTool(req *Request) *Response {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, InvalidParams, "Invalid params", err.Error())
	}

	var result *CallToolResult

	switch params.Name {
	case "cave_create":
		result = s.handleCaveCreate(params.Arguments)
	case "cave_list":
		result = s.handleCaveList(params.Arguments)
	case "cave_get":
		result = s.handleCaveGet(params.Arguments)
	case "cave_activate":
		result = s.handleCaveActivate(params.Arguments)
	case "cave_deactivate":
		result = s.handleCaveDeactivate(params.Arguments)
	case "cave_remove":
		result = s.handleCaveRemove(params.Arguments)
	case "workspace_info":
		result = s.handleWorkspaceInfo(params.Arguments)
	default:
		return NewErrorResponse(req.ID, MethodNotFound, "Unknown tool", nil)
	}

	return NewResponse(req.ID, result)
}

// Tool handlers

func (s *Server) handleCaveCreate(args json.RawMessage) *CallToolResult {
	var params struct {
		Name        string `json:"name"`
		BaseBranch  string `json:"baseBranch,omitempty"`
		AgentID     string `json:"agentId,omitempty"`
		Description string `json:"description,omitempty"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return NewToolError(fmt.Errorf("invalid parameters: %w", err))
	}

	opts := workspace.CreateOptions{
		Name:        params.Name,
		BaseBranch:  params.BaseBranch,
		AgentID:     params.AgentID,
		Description: params.Description,
	}

	ws, err := s.workspaceManager.Create(opts)
	if err != nil {
		return NewToolError(err)
	}

	result, _ := json.MarshalIndent(ws, "", "  ")
	return &CallToolResult{
		Content: NewToolContent(string(result)),
	}
}

func (s *Server) handleCaveList(args json.RawMessage) *CallToolResult {
	var params struct {
		Status string `json:"status,omitempty"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return NewToolError(fmt.Errorf("invalid parameters: %w", err))
	}

	opts := workspace.ListOptions{}
	if params.Status != "" {
		opts.Status = workspace.Status(params.Status)
	}

	workspaces, err := s.workspaceManager.List(opts)
	if err != nil {
		return NewToolError(err)
	}

	result, _ := json.MarshalIndent(workspaces, "", "  ")
	return &CallToolResult{
		Content: NewToolContent(string(result)),
	}
}

func (s *Server) handleCaveGet(args json.RawMessage) *CallToolResult {
	var params struct {
		CaveID string `json:"cave_id"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return NewToolError(fmt.Errorf("invalid parameters: %w", err))
	}

	ws, err := s.workspaceManager.Get(params.CaveID)
	if err != nil {
		return NewToolError(err)
	}

	result, _ := json.MarshalIndent(ws, "", "  ")
	return &CallToolResult{
		Content: NewToolContent(string(result)),
	}
}

func (s *Server) handleCaveActivate(args json.RawMessage) *CallToolResult {
	var params struct {
		CaveID string `json:"cave_id"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return NewToolError(fmt.Errorf("invalid parameters: %w", err))
	}

	if err := s.workspaceManager.Activate(params.CaveID); err != nil {
		return NewToolError(err)
	}

	return &CallToolResult{
		Content: NewToolContent(fmt.Sprintf("Workspace %s activated", params.CaveID)),
	}
}

func (s *Server) handleCaveDeactivate(args json.RawMessage) *CallToolResult {
	var params struct {
		CaveID string `json:"cave_id"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return NewToolError(fmt.Errorf("invalid parameters: %w", err))
	}

	if err := s.workspaceManager.Deactivate(params.CaveID); err != nil {
		return NewToolError(err)
	}

	return &CallToolResult{
		Content: NewToolContent(fmt.Sprintf("Workspace %s deactivated", params.CaveID)),
	}
}

func (s *Server) handleCaveRemove(args json.RawMessage) *CallToolResult {
	var params struct {
		CaveID string `json:"cave_id"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return NewToolError(fmt.Errorf("invalid parameters: %w", err))
	}

	if err := s.workspaceManager.Remove(params.CaveID); err != nil {
		return NewToolError(err)
	}

	return &CallToolResult{
		Content: NewToolContent(fmt.Sprintf("Workspace %s removed", params.CaveID)),
	}
}

func (s *Server) handleWorkspaceInfo(args json.RawMessage) *CallToolResult {
	var params struct {
		CaveID string `json:"cave_id"`
		Path   string `json:"path,omitempty"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return NewToolError(fmt.Errorf("invalid parameters: %w", err))
	}

	// Get workspace
	ws, err := s.workspaceManager.Get(params.CaveID)
	if err != nil {
		return NewToolError(err)
	}

	// Validate path
	if err := git.ValidateWorktreePath(ws.Path, params.Path); err != nil {
		return NewToolError(fmt.Errorf("invalid path: %w", err))
	}

	fullPath := filepath.Join(ws.Path, params.Path)
	
	// Check if path exists
	info, err := os.Stat(fullPath)
	if err != nil {
		return NewToolError(fmt.Errorf("path not found: %w", err))
	}

	if info.IsDir() {
		// List directory contents
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return NewToolError(fmt.Errorf("failed to read directory: %w", err))
		}

		var files []map[string]interface{}
		for _, entry := range entries {
			fileInfo := map[string]interface{}{
				"name":  entry.Name(),
				"type":  "file",
				"size":  0,
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
			"path":  params.Path,
			"files": files,
		}, "", "  ")

		return &CallToolResult{
			Content: NewToolContent(string(result)),
		}
	}

	// Read file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return NewToolError(fmt.Errorf("failed to read file: %w", err))
	}

	// Limit file size
	const maxFileSize = 1024 * 1024 // 1MB
	if len(content) > maxFileSize {
		content = content[:maxFileSize]
		content = append(content, []byte("\n\n[File truncated - exceeds 1MB limit]")...)
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"type":    "file",
		"path":    params.Path,
		"size":    info.Size(),
		"content": string(content),
	}, "", "  ")

	return &CallToolResult{
		Content: NewToolContent(string(result)),
	}
}

// writeResponse writes a response to the writer
func (s *Server) writeResponse(w *bufio.Writer, resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	if _, err := w.Write(data); err != nil {
		return err
	}

	if _, err := w.WriteString("\n"); err != nil {
		return err
	}

	return w.Flush()
}