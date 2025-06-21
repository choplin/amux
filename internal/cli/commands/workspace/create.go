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
	Long: `Create a new isolated workspace for development.

Examples:
  # Create workspace with auto-generated branch name
  amux ws create fix-auth

  # Create workspace with new branch
  amux ws create fix-auth -b feature/auth-fix

  # Create workspace from existing branch
  amux ws create fix-auth -c feature/existing-work

  # Create workspace with new branch from specific base
  amux ws create fix-auth --base develop -b feature/auth-fix`,
	Args: cobra.ExactArgs(1),
	RunE: runCreateWorkspace,
}

func runCreateWorkspace(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate mutually exclusive flags
	if createBranch != "" && createCheckout != "" {
		return fmt.Errorf("cannot specify both --branch (-b) and --checkout (-c) flags")
	}

	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	opts := workspace.CreateOptions{
		Name:        name,
		BaseBranch:  createBaseBranch,
		Description: createDescription,
		BranchMode:  workspace.BranchModeCreate, // Default to create mode
	}

	// Set Branch field based on which flag was used
	if createBranch != "" {
		opts.Branch = createBranch
		opts.BranchMode = workspace.BranchModeCreate // Explicit: create new branch
	} else if createCheckout != "" {
		opts.Branch = createCheckout
		opts.BranchMode = workspace.BranchModeCheckout // Explicit: use existing branch
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
