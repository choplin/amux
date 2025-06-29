// Package session provides session management commands for amux
package session

import (
	"fmt"
	"os"

	"github.com/aki/amux/internal/config"
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

	// Attach to session
	if err := sessionMgr.Attach(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to attach to session: %w", err)
	}

	return nil
}
