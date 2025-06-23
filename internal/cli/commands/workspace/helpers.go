package workspace

import (
	"fmt"
	"log/slog"

	"github.com/aki/amux/internal/core/agent"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/session"
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
	// Get ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return fmt.Errorf("failed to create ID mapper: %w", err)
	}

	// Create agent manager
	agentManager := agent.NewManager(configManager)

	// Create logger
	log := logger.Default()

	// Create session manager
	sessionManager, err := session.NewManager(configManager.GetAmuxDir(), wsManager, agentManager, idMapper, session.WithLogger(log))
	if err != nil {
		return fmt.Errorf("failed to create session manager: %w", err)
	}

	// Create session checker adapter
	sessionChecker := session.NewWorkspaceSessionChecker(sessionManager)

	// Create session stopper adapter
	sessionStopper := session.NewSessionStopperAdapter(sessionManager)

	// Initialize semaphore in workspace manager
	wsManager.InitializeSemaphore(sessionChecker, sessionStopper, slog.Default())

	return nil
}
