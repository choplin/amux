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
	var keepWorkspace bool

	cmd := &cobra.Command{
		Use:     "remove <session>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a stopped session",
		Long: `Remove a stopped session from the session list.

This command permanently removes session metadata. Only stopped sessions
can be removed. To stop a running session first, use 'amux session stop'.

If the session created its workspace automatically, the workspace will also
be removed unless --keep-workspace is specified.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return removeSession(cmd, args, keepWorkspace)
		},
	}

	cmd.Flags().BoolVar(&keepWorkspace, "keep-workspace", false, "Keep auto-created workspace when removing session")

	return cmd
}

func removeSession(cmd *cobra.Command, args []string, keepWorkspace bool) error {
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
	sess, err := sessionManager.ResolveSession(session.Identifier(sessionID))
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is running
	if sess.Status().IsRunning() {
		return fmt.Errorf("cannot remove running session %s (use 'amux session stop' first)", sessionID)
	}

	// Get session info to get workspace ID
	sessionInfo := sess.Info()
	workspaceID := sessionInfo.WorkspaceID

	// Remove the session
	if err := sessionManager.Remove(session.ID(sess.ID())); err != nil {
		return fmt.Errorf("failed to remove session: %w", err)
	}

	ui.Success("Session %s removed", sessionID)

	// Check if workspace was auto-created and --keep-workspace was not specified
	if !keepWorkspace && workspaceID != "" {
		// Get workspace to check if it was auto-created
		ws, err := wsManager.ResolveWorkspace(workspace.Identifier(workspaceID))
		if err != nil {
			// Workspace might already be removed or not found, skip auto-removal
			return nil //nolint:nilerr // Workspace not found is not an error in this context
		}

		// If workspace was auto-created, try to remove it
		if ws.AutoCreated {
			// Check if any other sessions are using this workspace
			sessions, err := sessionManager.ListSessions()
			if err != nil {
				ui.Warning("Failed to check for other sessions using workspace: %v", err)
				return nil //nolint:nilerr // Continue even if we can't check other sessions
			}

			// Check if any other session is using the same workspace
			workspaceInUse := false
			for _, otherSession := range sessions {
				if otherSession.Info().WorkspaceID == workspaceID {
					workspaceInUse = true
					break
				}
			}

			if !workspaceInUse {
				// Remove the workspace
				if err := wsManager.Remove(workspace.Identifier(workspaceID)); err != nil {
					ui.Warning("Failed to remove auto-created workspace %s: %v", ws.Name, err)
				} else {
					ui.Success("Removed auto-created workspace: '%s'", ws.Name)
				}
			} else {
				ui.Info("Auto-created workspace not removed (still in use by other sessions)")
			}
		}
	}

	return nil
}
