package session

import (
	"fmt"
	"os"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/config"
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

	// Remove session
	if err := sessionMgr.Remove(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to remove session: %w", err)
	}

	ui.Success("Session removed: %s", sessionID)
	return nil
}
