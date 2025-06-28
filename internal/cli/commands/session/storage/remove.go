package storage

import (
	"context"
	"fmt"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/storage"
	"github.com/spf13/cobra"
)

var removeRecursive bool

var removeCmd = &cobra.Command{
	Use:     "rm <session-id> <path>",
	Aliases: []string{"remove"},
	Short:   "Remove a file or directory from session storage",
	Long:    "Remove a file or directory from session storage. Use -r to remove directories recursively.",
	Args:    cobra.ExactArgs(2),
	RunE:    runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeRecursive, "recursive", "r", false, "Remove directories recursively")
}

func runRemove(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get session manager
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}
	manager, err := session.SetupManager(projectRoot)
	if err != nil {
		return err
	}

	// Resolve session
	sess, err := manager.ResolveSession(ctx, session.Identifier(args[0]))
	if err != nil {
		return fmt.Errorf("failed to resolve session: %w", err)
	}

	// Create storage manager
	storageManager := storage.NewManager(sess)

	// Remove the path
	path := args[1]
	result, err := storageManager.Remove(ctx, path, removeRecursive)
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
