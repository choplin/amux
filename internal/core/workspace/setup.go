// Package workspace provides management of isolated git worktree-based development environments.
package workspace

import (
	"fmt"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
)

// SetupManager creates a fully configured workspace manager with all dependencies initialized.
// This is the primary entry point for creating a workspace manager in production code.
func SetupManager(projectRoot string) (*Manager, error) {
	// Initialize config manager
	configManager := config.NewManager(projectRoot)

	// Initialize workspace ID mapper
	idMapper, err := idmap.NewWorkspaceIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize workspace ID mapper: %w", err)
	}

	// Create workspace manager with workspace-specific ID mapper
	manager, err := NewManagerWithIDMapper(configManager, idMapper)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	return manager, nil
}
