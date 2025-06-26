package session

import (
	"context"
	"fmt"

	"github.com/aki/amux/internal/core/agent"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

// GetSessionManager creates a session manager with all dependencies.
// This is used by the storage subpackage to avoid import cycles.
func GetSessionManager() (*session.Manager, error) {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return nil, err
	}

	// Use the setup function to create a properly configured session manager
	return session.SetupManager(projectRoot)
}

// GetManagers creates both session and workspace managers with all dependencies.
// This avoids creating the managers twice when both are needed.
func GetManagers() (*session.Manager, *workspace.Manager, error) {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return nil, nil, err
	}

	// Create workspace manager first
	workspaceManager, err := workspace.SetupManager(projectRoot)
	if err != nil {
		return nil, nil, err
	}

	// Create session manager using the same workspace manager
	sessionManager, err := session.SetupManagerWithWorkspace(projectRoot, workspaceManager)
	if err != nil {
		return nil, nil, err
	}

	return sessionManager, workspaceManager, nil
}

// GetAllManagers creates all managers with all dependencies.
// This is used when multiple managers are needed.
func GetAllManagers() (*session.Manager, *workspace.Manager, *agent.Manager, error) {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return nil, nil, nil, err
	}

	// Create workspace manager first
	workspaceManager, err := workspace.SetupManager(projectRoot)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create session manager using the same workspace manager
	sessionManager, err := session.SetupManagerWithWorkspace(projectRoot, workspaceManager)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create config manager for agent manager
	configManager := config.NewManager(projectRoot)

	// Create agent manager
	agentManager := agent.NewManager(configManager)

	return sessionManager, workspaceManager, agentManager, nil
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
