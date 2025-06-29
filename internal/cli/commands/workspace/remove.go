package workspace

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/config"
	"github.com/aki/amux/internal/workspace"
)

var removeWorkspaceCmd = &cobra.Command{
	Use:     "remove <workspace-name-or-id>",
	Aliases: []string{"rm"},
	Short:   "Remove a workspace by name or ID",
	Args:    cobra.ExactArgs(1),
	RunE:    runRemoveWorkspace,
}

func runRemoveWorkspace(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}
	manager, err := workspace.SetupManager(projectRoot)
	if err != nil {
		return err
	}

	// Resolve workspace by name or ID
	ws, err := manager.ResolveWorkspace(cmd.Context(), workspace.Identifier(identifier))
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Check for active sessions
	sessionIDs, err := ws.SessionIDs()
	if err != nil {
		// If we can't check sessions, fall back to normal removal with warning
		ui.Warning("Could not check for active sessions: %v", err)
	} else if len(sessionIDs) > 0 && !removeForce {
		// Show error with session information
		ui.Error("Cannot remove workspace '%s' - currently in use by %d session(s):", ws.Name, len(sessionIDs))

		// TODO: Show detailed session information once we have session manager in CLI
		// For now, just show session IDs
		for _, sessionID := range sessionIDs {
			ui.OutputLine("  - %s", sessionID)
		}

		ui.OutputLine("")
		ui.OutputLine("Use --force to remove anyway, or stop the sessions first")
		return fmt.Errorf("workspace in use")
	}

	// Get current working directory for safety check
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if !removeForce {
		ui.Warning("This will remove workspace '%s' (%s) and its branch '%s'", ws.Name, ws.ID, ws.Branch)
		response := ui.Prompt("Are you sure? (y/N): ")
		if response != "y" && response != "Y" {
			ui.OutputLine("Removal cancelled")
			return nil
		}
	}

	if err := manager.Remove(cmd.Context(), workspace.Identifier(ws.ID), workspace.RemoveOptions{
		NoHooks:    removeNoHooks,
		CurrentDir: cwd,
	}); err != nil {
		return fmt.Errorf("failed to remove workspace: %w", err)
	}

	ui.Success("Workspace removed successfully: %s (%s)", ws.Name, ws.ID)

	return nil
}
