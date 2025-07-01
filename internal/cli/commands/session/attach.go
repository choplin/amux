// Package session provides session management commands for amux
package session

import (
	"fmt"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <session-id>",
	Short: "Attach to a running session",
	Long: `Attach to a running session.

This is only supported for tmux runtime sessions.`,
	Args: cobra.ExactArgs(1),
	RunE: AttachSession,
}

// AttachSession implements the session attach command
func AttachSession(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	sessionID := args[0]

	// Setup managers with project root detection
	_, sessionMgr, err := setupManagers()
	if err != nil {
		return err
	}

	// Get session details for output
	session, err := sessionMgr.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session '%s' not found. Run 'amux ps' to see active sessions", sessionID)
	}

	// Show attachment info
	ui.OutputLine("Attaching to session '%s' (runtime: %s)", session.ID, session.Runtime)

	// Attach to session
	if err := sessionMgr.Attach(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to attach to session '%s': %w", sessionID, err)
	}

	return nil
}
