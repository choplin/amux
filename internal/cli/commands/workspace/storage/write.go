package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/storage"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/spf13/cobra"
)

var writeCmd = &cobra.Command{
	Use:   "write <workspace-id> <path> [content]",
	Short: "Write a file to workspace storage",
	Long: `Write content to a file in workspace storage.

If content is provided as an argument, it will be written to the file.
If no content is provided, content will be read from stdin.

Examples:
  # Write content directly
  amux ws storage write 1 notes.txt "My notes"

  # Write from stdin
  echo "My notes" | amux ws storage write 1 notes.txt

  # Write from file
  cat input.txt | amux ws storage write 1 output.txt`,
	Args: cobra.RangeArgs(2, 3),
	RunE: runWrite,
}

func runWrite(cmd *cobra.Command, args []string) error {
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
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("workspace not found: %s", args[0])
		}
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Get content
	var content []byte
	if len(args) > 2 {
		// Content provided as argument
		content = []byte(args[2])
	} else {
		// Read from stdin
		content, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
	}

	// Create storage manager
	storageManager := storage.NewManager()

	// Write the file
	path := args[1]
	if err := storageManager.WriteFile(ctx, ws.StoragePath, path, content); err != nil {
		return err
	}

	ui.Success("Successfully wrote %d bytes to %s", len(content), path)

	return nil
}
