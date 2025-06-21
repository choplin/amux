package workspace

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/workspace"
)

var pruneWorkspaceCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove old idle workspaces",
	RunE:  runPruneWorkspace,
}

func runPruneWorkspace(cmd *cobra.Command, args []string) error {
	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	opts := workspace.CleanupOptions{
		Days:   pruneDays,
		DryRun: pruneDryRun,
	}

	removed, err := manager.Cleanup(cmd.Context(), opts)
	if err != nil {
		return fmt.Errorf("failed to prune workspaces: %w", err)
	}

	if len(removed) == 0 {
		ui.OutputLine("No workspaces to prune")
		return nil
	}

	if pruneDryRun {
		ui.OutputLine("Would remove %d workspace(s):", len(removed))
	} else {
		ui.OutputLine("Removed %d workspace(s)", len(removed))
	}

	for _, id := range removed {
		ui.OutputLine("  - %s", id)
	}

	return nil
}
