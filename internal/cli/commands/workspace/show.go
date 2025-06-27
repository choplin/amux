package workspace

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
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

	// Handle JSON output
	if ui.GlobalFormatter.IsJSON() {
		return ui.GlobalFormatter.Output(ws)
	}

	// Print detailed workspace information
	ui.PrintWorkspaceDetails(ws)

	return nil
}
