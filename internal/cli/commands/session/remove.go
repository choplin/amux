package session

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

func removeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <session>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a stopped session",
		Long: `Remove a stopped session from the session list.

This command permanently removes session metadata. Only stopped sessions
can be removed. To stop a running session first, use 'amux session stop'.`,
		Args: cobra.ExactArgs(1),
		RunE: removeSession,
	}

	return cmd
}

func removeSession(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	// Create managers
	configManager := config.NewManager(projectRoot)
	wsManager, err := workspace.NewManager(configManager)
	if err != nil {
		return fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Create session manager
	sessionManager, err := createSessionManager(configManager, wsManager)
	if err != nil {
		return err
	}

	// Get session to check its status
	sess, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is running
	if sess.Status() == session.StatusRunning {
		return fmt.Errorf("cannot remove running session %s (use 'amux session stop' first)", sessionID)
	}

	// Remove the session
	if err := sessionManager.RemoveSession(sessionID); err != nil {
		return fmt.Errorf("failed to remove session: %w", err)
	}

	ui.Success("Session %s removed", sessionID)
	return nil
}
