package session

import (
	"fmt"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/mailbox"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

// createSessionManager is a helper to create a session manager with all dependencies
func createSessionManager(configManager *config.Manager, wsManager *workspace.Manager) (*session.Manager, error) {
	// Get ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return nil, fmt.Errorf("failed to create ID mapper: %w", err)
	}

	store, err := session.NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}

	// Create mailbox manager
	mailboxManager := mailbox.NewManager(configManager.GetAmuxDir())

	// Create logger
	log := logger.Default()

	return session.NewManager(store, wsManager, mailboxManager, idMapper, session.WithLogger(log)), nil
}

// createAutoWorkspace creates a new workspace with a name based on session ID
func createAutoWorkspace(wsManager *workspace.Manager, sessionID session.ID) (*workspace.Workspace, error) {
	// Use first 8 chars of session ID for workspace name
	name := fmt.Sprintf("session-%s", sessionID.Short())

	// Create the workspace
	opts := workspace.CreateOptions{
		Name:        name,
		Description: fmt.Sprintf("Auto-created workspace for session %s", sessionID.Short()),
		BaseBranch:  "main",
		AutoCreated: true,
	}

	ws, err := wsManager.Create(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	return ws, nil
}
