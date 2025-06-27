package session

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
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

	// Get managers
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}
	sessionManager, err := session.SetupManager(projectRoot)
	if err != nil {
		return err
	}

	// Get session
	sess, err := sessionManager.ResolveSession(cmd.Context(), session.Identifier(sessionID))
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Stop session with hook execution
	if err := sessionManager.StopSession(cmd.Context(), sess, false); err != nil {
		return fmt.Errorf("failed to stop session: %w", err)
	}

	ui.OutputLine("Session %s stopped", sessionID)
	return nil
}
