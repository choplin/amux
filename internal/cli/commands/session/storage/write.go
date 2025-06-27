package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
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

	// Get session info
	info := sess.Info()
	if info.StoragePath == "" {
		return fmt.Errorf("storage path not found for session")
	}

	// Construct full path
	path := args[1]
	fullPath := filepath.Join(info.StoragePath, path)

	// Ensure the path is within the storage directory
	cleanPath := filepath.Clean(fullPath)
	cleanStoragePath := filepath.Clean(info.StoragePath)
	if !strings.HasPrefix(cleanPath, cleanStoragePath) {
		return fmt.Errorf("path traversal attempt detected")
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

	// Create directory if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	ui.Success("Successfully wrote %d bytes to %s", len(content), path)

	return nil
}
