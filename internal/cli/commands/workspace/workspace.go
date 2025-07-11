// Package workspace provides CLI commands for managing amux workspaces.
package workspace

import (
	"github.com/aki/amux/internal/cli/commands/workspace/storage"
	"github.com/spf13/cobra"
)

var (
	// Create flags
	createBaseBranch  string
	createBranch      string // Create new branch with specified name
	createCheckout    string // Checkout existing branch
	createDescription string
	createNoHooks     bool

	// List flags
	listOneline bool

	// Prune flags
	pruneDays   int
	pruneDryRun bool

	// Remove flags
	removeForce   bool
	removeNoHooks bool
)

var workspaceCmd = &cobra.Command{
	Use:     "workspace",
	Aliases: []string{"ws"},
	Short:   "Manage Amux workspaces",
	Long:    "Create, list, and remove isolated development workspaces",
}

func init() {
	// Add subcommands
	workspaceCmd.AddCommand(createWorkspaceCmd)
	workspaceCmd.AddCommand(listWorkspaceCmd)
	workspaceCmd.AddCommand(showWorkspaceCmd)
	workspaceCmd.AddCommand(removeWorkspaceCmd)
	workspaceCmd.AddCommand(pruneWorkspaceCmd)
	workspaceCmd.AddCommand(cdWorkspaceCmd)
	workspaceCmd.AddCommand(storage.Command())

	// Create command flags
	createWorkspaceCmd.Flags().StringVar(&createBaseBranch, "base", "", "Base branch for new branches")
	createWorkspaceCmd.Flags().StringVarP(&createBranch, "branch", "b", "", "Create new branch with specified name")
	createWorkspaceCmd.Flags().StringVarP(&createCheckout, "checkout", "c", "", "Use existing branch")
	createWorkspaceCmd.Flags().StringVarP(&createDescription, "description", "d", "", "Description of the workspace")
	createWorkspaceCmd.Flags().BoolVar(&createNoHooks, "no-hooks", false, "Skip running hooks for this operation")

	// List command flags
	listWorkspaceCmd.Flags().BoolVar(&listOneline, "oneline", false, "Show one workspace per line (for use with fzf)")

	// Prune command flags
	pruneWorkspaceCmd.Flags().IntVarP(&pruneDays, "days", "d", 7, "Remove workspaces idle for more than N days")
	pruneWorkspaceCmd.Flags().BoolVar(&pruneDryRun, "dry-run", false, "Show what would be removed without removing")

	// Remove command flags
	removeWorkspaceCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Force removal without confirmation")
	removeWorkspaceCmd.Flags().BoolVar(&removeNoHooks, "no-hooks", false, "Skip running hooks for this operation")
}

// Command returns the workspace command
func Command() *cobra.Command {
	return workspaceCmd
}
