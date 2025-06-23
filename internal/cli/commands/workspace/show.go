package workspace

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/workspace"
)

var showWorkspaceCmd = &cobra.Command{
	Use:   "show <workspace-name-or-id>",
	Short: "Show detailed information about a workspace",
	Args:  cobra.ExactArgs(1),
	RunE:  runShowWorkspace,
}

func runShowWorkspace(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	// Resolve workspace by name or ID
	ws, err := manager.ResolveWorkspace(cmd.Context(), workspace.Identifier(identifier))
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Handle JSON output
	if ui.GlobalFormatter.IsJSON() {
		return ui.GlobalFormatter.Output(ws)
	}

	// Print detailed workspace information
	ui.PrintWorkspaceDetails(ws)

	// Display semaphore information from workspace
	if ws.IsInUse() {
		ui.OutputLine("\n%s Active Sessions (%d):\n", ui.InfoIcon, ws.GetHolderCount())
		for _, holder := range ws.SemaphoreHolders {
			description := holder.Description
			if description == "" {
				description = fmt.Sprintf("Session %s", holder.SessionID)
			}
			ui.OutputLine("  - %s: %s (%s ago)", holder.ID, description, ui.FormatDuration(time.Since(holder.Timestamp)))
		}
	} else {
		ui.OutputLine("\n%s No active sessions", ui.SuccessIcon)
	}

	return nil
}
