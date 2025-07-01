package session

import (
	"fmt"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <session-id>",
	Short: "Remove a stopped session",
	Long:  `Remove a stopped session from the session list.`,
	Args:  cobra.ExactArgs(1),
	RunE:  RemoveSession,
}

// RemoveSession implements the session remove command
func RemoveSession(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	sessionID := args[0]

	// Setup managers with project root detection
	_, sessionMgr, err := setupManagers()
	if err != nil {
		return err
	}

	// Remove session
	if err := sessionMgr.Remove(ctx, sessionID); err != nil {
		// Check if session not found
		if _, getErr := sessionMgr.Get(ctx, sessionID); getErr != nil {
			return fmt.Errorf("session '%s' not found. Run 'amux ps' to see active sessions", sessionID)
		}
		return fmt.Errorf("failed to remove session '%s': %w", sessionID, err)
	}

	ui.Success("Session removed: %s", sessionID)
	return nil
}
