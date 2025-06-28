package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/storage"
	"github.com/spf13/cobra"
)

var writeCmd = &cobra.Command{
	Use:   "write <session-id> <path> [content]",
	Short: "Write a file to session storage",
	Long: `Write content to a file in session storage.

If content is provided as an argument, it will be written to the file.
If no content is provided, content will be read from stdin.

Examples:
  # Write content directly
  amux session storage write 1 output.log "Test output"

  # Write from stdin
  echo "Test output" | amux session storage write 1 output.log

  # Write from file
  cat input.txt | amux session storage write 1 output.txt`,
	Args: cobra.RangeArgs(2, 3),
	RunE: runWrite,
}

func runWrite(cmd *cobra.Command, args []string) error {
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
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("session not found: %s", args[0])
		}
		return fmt.Errorf("failed to resolve session: %w", err)
	}

	// Create storage manager
	storageManager := storage.NewManager(sess)

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

	// Write the file
	path := args[1]
	if err := storageManager.WriteFile(ctx, path, content); err != nil {
		return err
	}

	ui.Success("Successfully wrote %d bytes to %s", len(content), path)

	return nil
}
