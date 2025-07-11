package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/aki/amux/internal/config"
	"github.com/aki/amux/internal/storage"
	"github.com/aki/amux/internal/workspace"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <workspace-id> <path>",
	Short: "Read a file from workspace storage",
	Long:  "Read and display the contents of a file from workspace storage",
	Args:  cobra.ExactArgs(2),
	RunE:  runRead,
}

func runRead(cmd *cobra.Command, args []string) error {
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
	storageManager := storage.NewManager(ws)

	// Read the file
	path := args[1]
	content, err := storageManager.ReadFile(ctx, path)
	if err != nil {
		var notFound storage.ErrNotFound
		if errors.As(err, &notFound) {
			return fmt.Errorf("file not found: %s", path)
		}
		return err
	}

	// Display the content
	fmt.Print(string(content))

	// If content doesn't end with newline, add one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		fmt.Println()
	}

	return nil
}
