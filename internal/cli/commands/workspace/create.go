package workspace

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/hooks"
	"github.com/aki/amux/internal/core/workspace"
)

var createWorkspaceCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new workspace",
	Args:  cobra.ExactArgs(1),
	RunE:  runCreateWorkspace,
}

func runCreateWorkspace(cmd *cobra.Command, args []string) error {
	name := args[0]

	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	opts := workspace.CreateOptions{
		Name:        name,
		BaseBranch:  createBaseBranch,
		Branch:      createBranch,
		Description: createDescription,
	}

	ws, err := manager.Create(cmd.Context(), opts)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	id := ws.ID
	if ws.Index != "" {
		id = ws.Index
	}
	ui.Success("Workspace created successfully")
	ui.OutputLine("")
	ui.PrintKeyValue("ID", id)
	ui.PrintKeyValue("Branch", ws.Branch)
	ui.PrintKeyValue("Path", ws.Path)

	// Execute hooks unless --no-hooks was specified
	if !createNoHooks {
		if err := executeWorkspaceHooks(ws, hooks.EventWorkspaceCreate); err != nil {
			// Hooks failed but workspace was created
			ui.Error("Hook execution failed: %v", err)
			ui.Warning("Workspace was created but hooks did not run successfully")
			return nil
		}
	}

	return nil
}
