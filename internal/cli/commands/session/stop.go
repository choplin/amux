package session

import (
	"fmt"
	"os"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/config"
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

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create config manager
	configMgr := config.NewManager(wd)
	if !configMgr.IsInitialized() {
		return fmt.Errorf("amux not initialized. Run 'amux init' first")
	}

	// Get session manager
	sessionMgr := getSessionManager(configMgr)

	// Stop or kill session
	if stopOpts.force {
		if err := sessionMgr.Kill(ctx, sessionID); err != nil {
			return fmt.Errorf("failed to kill session: %w", err)
		}
		ui.Success("Session killed: %s", sessionID)
	} else {
		if err := sessionMgr.Stop(ctx, sessionID); err != nil {
			return fmt.Errorf("failed to stop session: %w", err)
		}
		ui.Success("Session stopped: %s", sessionID)
	}

	return nil
}
