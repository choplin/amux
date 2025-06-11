package commands

import (
	"fmt"

	"github.com/aki/amux/internal/core/common"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/mailbox"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

// createSessionManager is a helper to create a session manager with all dependencies
func createSessionManager(configManager *config.Manager, wsManager *workspace.Manager, idMapper *common.IDMapper) (*session.Manager, error) {
	store, err := session.NewFileStore(configManager.GetAmuxDir())
	if err != nil {
		return nil, fmt.Errorf("failed to create session store: %w", err)
	}

	// Create mailbox manager
	mailboxManager := mailbox.NewManager(configManager.GetAmuxDir())

	return session.NewManager(store, wsManager, mailboxManager, idMapper), nil
}
