package commands

import (
	"fmt"

	"github.com/aki/amux/internal/core/common"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

// createManagers is a helper to create commonly used managers
func createManagers() (*config.Manager, *workspace.Manager, *common.IDMapper, error) {
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return nil, nil, nil, err
	}

	configManager := config.NewManager(projectRoot)
	if !configManager.IsInitialized() {
		return nil, nil, nil, fmt.Errorf("Amux not initialized. Run 'amux init' first")
	}

	wsManager, err := workspace.NewManager(configManager)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	idMapper, err := common.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create ID mapper: %w", err)
	}

	return configManager, wsManager, idMapper, nil
}

// createSessionManager is a helper to create a session manager with all dependencies
func createSessionManager(configManager *config.Manager, wsManager *workspace.Manager, idMapper *common.IDMapper) (*session.Manager, error) {
	store, err := session.NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}

	return session.NewManager(store, wsManager, idMapper), nil
}
