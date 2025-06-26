package storage

import (
	"github.com/aki/amux/internal/app"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
)

// getSessionManager creates a session manager for storage commands.
// This is a local helper to avoid import cycles with the parent package.
func getSessionManager() (*session.Manager, error) {
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
