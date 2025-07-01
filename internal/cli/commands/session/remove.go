package session

import (
	"fmt"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/session"
	"github.com/aki/amux/internal/workspace"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <session-id>",
	Short: "Remove a stopped session",
	Long: `Remove a stopped session from the session list.

This command permanently removes session metadata. Only stopped sessions
can be removed. To stop a running session first, use 'amux session stop'.

If the session created its workspace automatically, the workspace will also
be removed unless --keep-workspace is specified.

Use --force to automatically stop a running session before removal.`,
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"rm", "delete"},
	RunE:    RemoveSession,
}

var removeOpts struct {
	keepWorkspace bool
	force         bool
}

func init() {
	removeCmd.Flags().BoolVar(&removeOpts.keepWorkspace, "keep-workspace", false, "Keep auto-created workspace when removing session")
	removeCmd.Flags().BoolVarP(&removeOpts.force, "force", "f", false, "Force removal by stopping running sessions first")
}

// RemoveSession implements the session remove command
func RemoveSession(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	sessionID := args[0]

	// Setup managers with project root detection
	configMgr, sessionMgr, err := setupManagers()
	if err != nil {
		return err
	}

	// Get session to check its status and workspace
	sess, err := sessionMgr.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("session '%s' not found. Run 'amux ps' to see active sessions", sessionID)
	}

	// Check if session is running
	if sess.Status == session.StatusRunning {
		if !removeOpts.force {
			return fmt.Errorf("cannot remove running session %s (use 'amux session stop' first or --force)", sessionID)
		}
		// Force flag is set, stop the session first
		ui.Info("Stopping running session %s...", sessionID)
		if err := sessionMgr.Stop(ctx, sessionID); err != nil {
			return fmt.Errorf("failed to stop session: %w", err)
		}
		ui.Success("Session stopped successfully")
	}

	// Save workspace ID before removing session
	workspaceID := sess.WorkspaceID

	// Remove the session
	if err := sessionMgr.Remove(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to remove session '%s': %w", sessionID, err)
	}

	ui.Success("Session removed: %s", sessionID)

	// Check if workspace was auto-created and --keep-workspace was not specified
	if !removeOpts.keepWorkspace && workspaceID != "" {
		// Create workspace manager
		wsMgr, err := workspace.SetupManager(configMgr.GetProjectRoot())
		if err != nil {
			// If we can't create workspace manager, skip workspace removal
			return nil
		}

		// Get workspace to check if it was auto-created
		ws, err := wsMgr.Get(ctx, workspace.ID(workspaceID))
		if err != nil {
			// Workspace might already be removed or not found, skip auto-removal
			return nil
		}

		// If workspace was auto-created, try to remove it
		if ws.AutoCreated {
			// Check if any other sessions are using this workspace
			sessions, err := sessionMgr.List(ctx, "")
			if err != nil {
				ui.Warning("Failed to check for other sessions using workspace: %v", err)
				return nil
			}

			// Check if any other session is using the same workspace
			workspaceInUse := false
			for _, otherSession := range sessions {
				if otherSession.WorkspaceID == workspaceID {
					workspaceInUse = true
					break
				}
			}

			if !workspaceInUse {
				// Remove the workspace
				if err := wsMgr.Remove(ctx, workspace.Identifier(workspaceID), workspace.RemoveOptions{NoHooks: false}); err != nil {
					ui.Warning("Failed to remove auto-created workspace %s: %v", ws.Name, err)
				} else {
					ui.OutputLine("Removed auto-created workspace: '%s'", ws.Name)
				}
			} else {
				ui.Info("Auto-created workspace not removed (still in use by other sessions)")
			}
		}
	}

	return nil
}
