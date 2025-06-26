package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aki/amux/internal/cli/ui"
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
	manager, err := getWorkspaceManager()
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

	// Check if the path exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("failed to stat path: %w", err)
	}

	// Remove the file or directory
	if info.IsDir() {
		if !removeRecursive {
			return fmt.Errorf("cannot remove directory without -r flag: %s", path)
		}
		if err := os.RemoveAll(fullPath); err != nil {
			return fmt.Errorf("failed to remove directory: %w", err)
		}
		ui.Success("Removed directory: %s", path)
	} else {
		if err := os.Remove(fullPath); err != nil {
			return fmt.Errorf("failed to remove file: %w", err)
		}
		ui.Success("Removed file: %s", path)
	}

	return nil
}
