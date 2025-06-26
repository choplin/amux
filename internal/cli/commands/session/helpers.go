package session

import (
	"context"
	"fmt"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

// createSessionManager is a helper to create a session manager with all dependencies
func createSessionManager(configManager *config.Manager, wsManager *workspace.Manager) (*session.Manager, error) {
	factory := session.NewFactory()
	return factory.CreateManager(configManager, wsManager)
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
