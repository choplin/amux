package session

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/workspace"
)

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <session>",
		Short: "Stop a running session",
		Args:  cobra.ExactArgs(1),
		RunE:  stopSession,
	}
}

func stopSession(cmd *cobra.Command, args []string) error {
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

	// Get session
	sess, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Stop session
	if err := sess.Stop(); err != nil {
		return fmt.Errorf("failed to stop session: %w", err)
	}

	ui.Success("Session %s stopped", sessionID)
	return nil
}
