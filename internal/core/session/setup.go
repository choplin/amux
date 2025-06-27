// Package session provides management of agent execution sessions.
package session

import (
	"fmt"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/workspace"
)

// SetupManager creates a fully configured session manager with all dependencies initialized.
// This is the primary entry point for creating a session manager in production code.
func SetupManager(projectRoot string) (*Manager, error) {
	// Initialize config manager
	configManager := config.NewManager(projectRoot)

	// Initialize session ID mapper
	idMapper, err := idmap.NewSessionIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize session ID mapper: %w", err)
	}

	// Initialize workspace manager independently
	workspaceManager, err := workspace.SetupManager(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Create session manager
	sessionManager, err := NewManager(
		configManager.GetAmuxDir(),
		workspaceManager,
		configManager,
		idMapper,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	return sessionManager, nil
}
