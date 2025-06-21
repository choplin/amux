package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/hooks"
	"github.com/aki/amux/internal/core/workspace"
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

	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	// Resolve workspace by name or ID
	ws, err := manager.ResolveWorkspace(cmd.Context(), workspace.Identifier(identifier))
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Check if current working directory is inside the workspace
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Resolve current directory to handle OS-level symlinks (e.g., macOS /var -> /private/var)
	resolvedCwd, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		// If we can't resolve, use original path
		resolvedCwd = cwd
	}

	// Check both original and resolved paths
	if strings.HasPrefix(cwd, ws.Path) || strings.HasPrefix(resolvedCwd, ws.Path) {
		ui.Error("Cannot remove workspace while working inside it")
		ui.OutputLine("\nPlease change to a different directory first:")
		projectRoot, _ := config.FindProjectRoot()
		if projectRoot != "" {
			ui.OutputLine("  cd %s", projectRoot)
		}
		return fmt.Errorf("cannot remove workspace from within itself")
	}

	if !removeForce {
		ui.Warning("This will remove workspace '%s' (%s) and its branch '%s'", ws.Name, ws.ID, ws.Branch)
		response := ui.Prompt("Are you sure? (y/N): ")
		if response != "y" && response != "Y" {
			ui.OutputLine("Removal cancelled")
			return nil
		}
	}

	// Execute pre-removal hooks unless --no-hooks was specified
	if !removeNoHooks {
		if err := executeWorkspaceHooks(ws, hooks.EventWorkspaceRemove); err != nil {
			ui.Error("Hook execution failed: %v", err)
			ui.Warning("Stopping removal due to hook failure")
			return err
		}
	}

	if err := manager.Remove(cmd.Context(), workspace.Identifier(ws.ID)); err != nil {
		return fmt.Errorf("failed to remove workspace: %w", err)
	}

	ui.Success("Workspace removed successfully: %s (%s)", ws.Name, ws.ID)

	return nil
}
