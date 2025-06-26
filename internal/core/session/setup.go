// Package session provides management of agent execution sessions.
package session

import (
	"fmt"

	"github.com/aki/amux/internal/core/agent"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/workspace"
)

// SetupManager creates a fully configured session manager with all dependencies initialized.
// This is the primary entry point for creating a session manager in production code.
// If you need to share a WorkspaceManager instance, use SetupManagerWithWorkspace instead.
func SetupManager(projectRoot string) (*Manager, error) {
	// Initialize config manager
	configManager := config.NewManager(projectRoot)

	// Initialize ID mapper (shared between workspace and session managers)
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ID mapper: %w", err)
	}

	// Initialize workspace manager with the shared ID mapper
	workspaceManager, err := workspace.NewManagerWithIDMapper(configManager, idMapper)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Initialize agent manager
	agentManager := agent.NewManager(configManager)

	// Create session manager
	sessionManager, err := NewManager(
		configManager.GetAmuxDir(),
		workspaceManager,
		agentManager,
		idMapper,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	return sessionManager, nil
}

// SetupManagerWithWorkspace creates a session manager using an existing workspace manager.
// This allows sharing the workspace manager (and its ID mapper) between multiple components.
func SetupManagerWithWorkspace(projectRoot string, workspaceManager *workspace.Manager) (*Manager, error) {
	// Initialize config manager
	configManager := config.NewManager(projectRoot)

	// Initialize agent manager
	agentManager := agent.NewManager(configManager)

	// Get the ID mapper from workspace manager
	idMapper := workspaceManager.GetIDMapper()
	if idMapper == nil {
		return nil, fmt.Errorf("workspace manager has no ID mapper")
	}

	// Create session manager
	sessionManager, err := NewManager(
		configManager.GetAmuxDir(),
		workspaceManager,
		agentManager,
		idMapper,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	return sessionManager, nil
}
