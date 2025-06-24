package workspace

import (
	"fmt"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/workspace"
)

func getWorkspaceManager() (*workspace.Manager, error) {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return nil, err
	}

	// Create configuration manager
	configManager := config.NewManager(projectRoot)

	// Ensure initialized
	if !configManager.IsInitialized() {
		return nil, fmt.Errorf("amux not initialized. Run 'amux init' first")
	}

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	if err != nil {
		return nil, err
	}

	// Initialize semaphore support
	if err := initializeSemaphoreSupport(configManager, wsManager); err != nil {
		// Log error but continue - semaphore is optional
		logger.Default().Warn("failed to initialize semaphore support", "error", err)
	}

	return wsManager, nil
}

func initializeSemaphoreSupport(configManager *config.Manager, wsManager *workspace.Manager) error {
	// Semaphore is now built into workspace manager
	// No initialization needed
	return nil
}
