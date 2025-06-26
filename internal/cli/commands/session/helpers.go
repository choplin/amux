package session

import (
	"context"
	"fmt"

	"github.com/aki/amux/internal/app"
	"github.com/aki/amux/internal/core/agent"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

// GetSessionManager creates a session manager with all dependencies.
// This is the public version for use by subpackages.
func GetSessionManager() (*session.Manager, error) {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return nil, err
	}

	// Create container with all dependencies
	container, err := app.NewContainer(projectRoot)
	if err != nil {
		return nil, err
	}

	return container.SessionManager, nil
}

// GetManagers creates both session and workspace managers with all dependencies.
// This avoids creating the container twice when both managers are needed.
func GetManagers() (*session.Manager, *workspace.Manager, error) {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return nil, nil, err
	}

	// Create container with all dependencies
	container, err := app.NewContainer(projectRoot)
	if err != nil {
		return nil, nil, err
	}

	return container.SessionManager, container.WorkspaceManager, nil
}

// GetAllManagers creates all managers with all dependencies.
// This is used when multiple managers are needed.
func GetAllManagers() (*session.Manager, *workspace.Manager, *agent.Manager, error) {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return nil, nil, nil, err
	}

	// Create container with all dependencies
	container, err := app.NewContainer(projectRoot)
	if err != nil {
		return nil, nil, nil, err
	}

	return container.SessionManager, container.WorkspaceManager, container.AgentManager, nil
}

// createAutoWorkspace creates a new workspace with a name based on session ID
func createAutoWorkspace(ctx context.Context, wsManager *workspace.Manager, sessionID session.ID, customName, customDescription string) (*workspace.Workspace, error) {
	// Use custom name if provided, otherwise use session ID
	name := customName
	if name == "" {
		// Use first 8 chars of session ID for workspace name
		name = fmt.Sprintf("session-%s", sessionID.Short())
	}

	// Use custom description if provided, otherwise use default
	description := customDescription
	if description == "" {
		description = fmt.Sprintf("Auto-created workspace for session %s", sessionID.Short())
	}

	// Create the workspace
	opts := workspace.CreateOptions{
		Name:        name,
		Description: description,
		BaseBranch:  "main",
		AutoCreated: true,
	}

	ws, err := wsManager.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	return ws, nil
}
