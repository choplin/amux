package workspace

import (
	"github.com/aki/amux/internal/app"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/workspace"
)

// GetWorkspaceManager creates a workspace manager with all dependencies.
// This is the public version for use by subpackages.
func GetWorkspaceManager() (*workspace.Manager, error) {
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

	return container.WorkspaceManager, nil
}
