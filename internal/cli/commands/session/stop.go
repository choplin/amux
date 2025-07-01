package session

import (
	"fmt"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop <session-id>",
	Short: "Stop a running session",
	Long:  `Stop a running session gracefully.`,
	Args:  cobra.ExactArgs(1),
	RunE:  StopSession,
}

var stopOpts struct {
	force bool
}

func init() {
	stopCmd.Flags().BoolVarP(&stopOpts.force, "force", "f", false, "Force kill the session")
}

// StopSession implements the session stop command
func StopSession(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	sessionID := args[0]

	// Setup managers with project root detection
	_, sessionMgr, err := setupManagers()
	if err != nil {
		return err
	}

	// Stop or kill session
	if stopOpts.force {
		if err := sessionMgr.Kill(ctx, sessionID); err != nil {
			// Check if session not found
			if _, getErr := sessionMgr.Get(ctx, sessionID); getErr != nil {
				return fmt.Errorf("session '%s' not found. Run 'amux ps' to see active sessions", sessionID)
			}
			return fmt.Errorf("failed to kill session '%s': %w", sessionID, err)
		}
		ui.Success("Session killed: %s", sessionID)
	} else {
		if err := sessionMgr.Stop(ctx, sessionID); err != nil {
			// Check if session not found
			if _, getErr := sessionMgr.Get(ctx, sessionID); getErr != nil {
				return fmt.Errorf("session '%s' not found. Run 'amux ps' to see active sessions", sessionID)
			}
			return fmt.Errorf("failed to stop session '%s': %w", sessionID, err)
		}
		ui.Success("Session stopped: %s", sessionID)
	}

	return nil
}
