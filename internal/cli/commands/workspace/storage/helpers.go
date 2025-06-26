package storage

import (
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/workspace"
)

// getWorkspaceManager creates a workspace manager for storage commands.
// This is a local helper to avoid import cycles with the parent package.
func getWorkspaceManager() (*workspace.Manager, error) {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return nil, err
	}

	// Use the setup function to create a properly configured workspace manager
	return workspace.SetupManager(projectRoot)
}
