package storage

import (
	"context"
	"fmt"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/storage"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/spf13/cobra"
)

var removeRecursive bool

var removeCmd = &cobra.Command{
	Use:     "rm <workspace-id> <path>",
	Aliases: []string{"remove"},
	Short:   "Remove a file or directory from workspace storage",
	Long:    "Remove a file or directory from workspace storage. Use -r to remove directories recursively.",
	Args:    cobra.ExactArgs(2),
	RunE:    runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeRecursive, "recursive", "r", false, "Remove directories recursively")
}

func runRemove(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get workspace manager
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}
	manager, err := workspace.SetupManager(projectRoot)
	if err != nil {
		return err
	}

	// Resolve workspace
	ws, err := manager.ResolveWorkspace(ctx, workspace.Identifier(args[0]))
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Create storage manager
	storageManager := storage.NewManager()

	// Remove the path
	path := args[1]
	result, err := storageManager.Remove(ctx, ws.StoragePath, path, removeRecursive)
	if err != nil {
		return err
	}

	if result.IsDir {
		ui.Success("Removed directory: %s", path)
	} else {
		ui.Success("Removed file: %s", path)
	}

	return nil
}
