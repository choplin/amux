package workspace

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/workspace"
)

var listWorkspaceCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all workspaces",
	Long: `List all workspaces in the project.

Examples:
  # List workspaces with detailed view
  amux ws ls

  # List workspaces in oneline format for scripting
  amux ws ls --oneline

  # Use with fzf to select a workspace
  amux ws ls --oneline | fzf | cut -f1

  # Remove selected workspace with fzf
  amux ws rm $(amux ws ls --oneline | fzf | cut -f1)`,
	RunE: runListWorkspace,
}

func runListWorkspace(cmd *cobra.Command, args []string) error {
	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	workspaces, err := manager.List(cmd.Context(), workspace.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	// Get semaphore information for each workspace
	entries := make([]ui.WorkspaceListEntry, len(workspaces))
	for i, ws := range workspaces {
		holders, err := manager.GetSemaphoreHolders(ws.ID)
		holderCount := 0
		if err == nil {
			holderCount = len(holders)
		}
		entries[i] = ui.WorkspaceListEntry{
			Workspace:   ws,
			HolderCount: holderCount,
		}
	}

	// Handle JSON output
	if ui.GlobalFormatter.IsJSON() {
		return ui.GlobalFormatter.Output(workspaces)
	}

	// Handle pretty output
	if listOneline {
		// One line per workspace for fzf integration
		for _, ws := range workspaces {
			// Format: name<tab>id<tab>branch<tab>status<tab>path<tab>description
			description := ws.Description
			if description == "" {
				description = "-"
			}

			id := ws.ID
			if ws.Index != "" {
				id = ws.Index
			}

			ui.PrintTSV([][]string{{ws.Name, id, ws.Branch, ws.Status.String(), ws.Path, description}})
		}
	} else {
		ui.PrintWorkspaceListWithHolders(entries)
	}

	return nil
}
