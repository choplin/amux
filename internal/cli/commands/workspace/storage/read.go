package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/workspace"
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
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("workspace not found: %s", args[0])
		}
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	if ws.StoragePath == "" {
		return fmt.Errorf("storage path not found for workspace")
	}

	// Construct full path
	path := args[1]
	fullPath := filepath.Join(ws.StoragePath, path)

	// Ensure the path is within the storage directory
	cleanPath := filepath.Clean(fullPath)
	cleanStoragePath := filepath.Clean(ws.StoragePath)
	if !strings.HasPrefix(cleanPath, cleanStoragePath) {
		return fmt.Errorf("path traversal attempt detected")
	}

	// Read the file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Display the content
	fmt.Print(string(content))

	// If content doesn't end with newline, add one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		fmt.Println()
	}

	return nil
}
