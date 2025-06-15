package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/session"
)

// parseSessionURI extracts the session ID and subpath from a URI like amux://session/{id}
func parseSessionURI(uri string) (string, error) {
	// Remove the scheme
	path := strings.TrimPrefix(uri, "amux://")

	// Split the path
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] != "session" {
		return "", fmt.Errorf("invalid session URI: %s", uri)
	}

	sessionID := parts[1]
	if sessionID == "" {
		return "", fmt.Errorf("invalid session URI: missing session ID")
	}

	return sessionID, nil
}

// sessionInfo is the data structure for session list items
type sessionInfo struct {
	ID          string              `json:"id"`
	Index       string              `json:"index,omitempty"`
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description,omitempty"`
	WorkspaceID string              `json:"workspaceId"`
	AgentID     string              `json:"agentId"`
	Status      session.Status      `json:"status"`
	CreatedAt   string              `json:"createdAt"`
	StartedAt   string              `json:"startedAt,omitempty"`
	StoppedAt   string              `json:"stoppedAt,omitempty"`
	Resources   sessionResourceURIs `json:"resources"`
}

type sessionResourceURIs struct {
	Detail string `json:"detail"`
	Output string `json:"output"`
}

// sessionDetail is the full session information for detail resource
type sessionDetail struct {
	ID          string              `json:"id"`
	Index       string              `json:"index,omitempty"`
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description,omitempty"`
	WorkspaceID string              `json:"workspaceId"`
	AgentID     string              `json:"agentId"`
	Status      session.Status      `json:"status"`
	Command     string              `json:"command,omitempty"`
	Environment map[string]string   `json:"environment,omitempty"`
	PID         int                 `json:"pid,omitempty"`
	TmuxSession string              `json:"tmuxSession,omitempty"`
	CreatedAt   string              `json:"createdAt"`
	StartedAt   string              `json:"startedAt,omitempty"`
	StoppedAt   string              `json:"stoppedAt,omitempty"`
	Error       string              `json:"error,omitempty"`
	StoragePath string              `json:"storagePath,omitempty"`
	Resources   sessionResourceURIs `json:"resources"`
}

// registerSessionResources registers session-related MCP resources
func (s *ServerV2) registerSessionResources() error {
	// Static resource: amux://session
	sessionListRes := mcp.NewResource(
		"amux://session",
		"Session List",
		mcp.WithResourceDescription("List all active sessions with metadata"),
		mcp.WithMIMEType("application/json"),
	)
	s.mcpServer.AddResource(sessionListRes, s.handleSessionListResource)

	// Register session detail template
	sessionDetailTemplate := mcp.NewResourceTemplate(
		"amux://session/{id}",
		"Session Details",
		mcp.WithTemplateDescription("Get session details including workspace, agent, status, and creation time"),
		mcp.WithTemplateMIMEType("application/json"),
	)
	s.mcpServer.AddResourceTemplate(sessionDetailTemplate, s.handleSessionDetailResource)

	// Register session output template
	sessionOutputTemplate := mcp.NewResourceTemplate(
		"amux://session/{id}/output",
		"Session Output",
		mcp.WithTemplateDescription("Read current session output/logs"),
		mcp.WithTemplateMIMEType("text/plain"),
	)
	s.mcpServer.AddResourceTemplate(sessionOutputTemplate, s.handleSessionOutputResource)

	return nil
}

// handleSessionListResource handles amux://session
func (s *ServerV2) handleSessionListResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Create session manager to list sessions
	sessionManager, err := s.createSessionManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	sessions, err := sessionManager.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Update all session statuses in batch for better performance
	sessionManager.UpdateAllStatuses(sessions)

	sessionList := make([]sessionInfo, len(sessions))
	for i, sess := range sessions {

		info := sess.Info()
		sessionInfo := sessionInfo{
			ID:          info.ID,
			Index:       info.Index,
			Name:        info.Name,
			Description: info.Description,
			WorkspaceID: info.WorkspaceID,
			AgentID:     info.AgentID,
			Status:      info.StatusState.Status,
			CreatedAt:   info.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		if info.StartedAt != nil {
			sessionInfo.StartedAt = info.StartedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		if info.StoppedAt != nil {
			sessionInfo.StoppedAt = info.StoppedAt.Format("2006-01-02T15:04:05Z07:00")
		}

		// Add resource URIs
		sessionInfo.Resources.Detail = fmt.Sprintf("amux://session/%s", info.ID)
		sessionInfo.Resources.Output = fmt.Sprintf("amux://session/%s/output", info.ID)

		sessionList[i] = sessionInfo
	}

	jsonData, err := json.MarshalIndent(sessionList, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session list: %w", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// handleSessionDetailResource handles amux://session/{id}
func (s *ServerV2) handleSessionDetailResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Parse session ID from URI
	sessionID, err := parseSessionURI(request.Params.URI)
	if err != nil {
		return nil, err
	}

	// Create session manager
	sessionManager, err := s.createSessionManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	// Get session (supports ID, index, or name)
	sess, err := sessionManager.ResolveSession(session.Identifier(sessionID))
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Update status for running sessions before returning details
	if sess.Status().IsRunning() {
		// Try to update status if session supports terminal operations
		if terminalSess, ok := sess.(session.TerminalSession); ok {
			_ = terminalSess.UpdateStatus() // Ignore errors, use current status if update fails
		}
	}

	info := sess.Info()
	detail := sessionDetail{
		ID:          info.ID,
		Index:       info.Index,
		Name:        info.Name,
		Description: info.Description,
		WorkspaceID: info.WorkspaceID,
		AgentID:     info.AgentID,
		Status:      info.StatusState.Status,
		Command:     info.Command,
		Environment: info.Environment,
		PID:         info.PID,
		TmuxSession: info.TmuxSession,
		CreatedAt:   info.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Error:       info.Error,
		StoragePath: info.StoragePath,
	}

	if info.StartedAt != nil {
		detail.StartedAt = info.StartedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if info.StoppedAt != nil {
		detail.StoppedAt = info.StoppedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	// Add resource URIs
	detail.Resources.Detail = fmt.Sprintf("amux://session/%s", info.ID)
	detail.Resources.Output = fmt.Sprintf("amux://session/%s/output", info.ID)

	jsonData, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session detail: %w", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// handleSessionOutputResource handles amux://session/{id}/output
func (s *ServerV2) handleSessionOutputResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Parse session ID from URI
	sessionID, err := parseSessionURI(request.Params.URI)
	if err != nil {
		return nil, err
	}

	// Create session manager
	sessionManager, err := s.createSessionManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	// Get session (supports ID, index, or name)
	sess, err := sessionManager.ResolveSession(session.Identifier(sessionID))
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is running
	if !sess.Status().IsRunning() {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "text/plain",
				Text:     fmt.Sprintf("Session %s is not running (status: %s)", sessionID, sess.Status()),
			},
		}, nil
	}

	// Type assert to TerminalSession
	terminalSess, ok := sess.(session.TerminalSession)
	if !ok {
		return []mcp.ResourceContents{
			&mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "text/plain",
				Text:     "Session does not support terminal operations",
			},
		}, nil
	}

	// Get output (0 = all lines for resource access)
	output, err := terminalSess.GetOutput(0)
	if err != nil {
		return nil, fmt.Errorf("failed to get session output: %w", err)
	}

	return []mcp.ResourceContents{
		&mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "text/plain",
			Text:     string(output),
		},
	}, nil
}

// createSessionManager is a helper to create a session manager with all dependencies
func (s *ServerV2) createSessionManager() (*session.Manager, error) {
	// Use existing workspace manager
	workspaceManager := s.workspaceManager

	// Create ID mapper
	idMapper, err := idmap.NewIDMapper(s.configManager.GetAmuxDir())
	if err != nil {
		return nil, fmt.Errorf("failed to create ID mapper: %w", err)
	}

	// Create session store
	store, err := session.NewFileStore(s.configManager.GetAmuxDir())
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}

	// Create session manager
	return session.NewManager(store, workspaceManager, idMapper), nil
}
