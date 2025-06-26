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

var listCmd = &cobra.Command{
	Use:   "list <workspace-id> [path]",
	Short: "List files in workspace storage",
	Long:  "List files and directories in the workspace storage directory",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Determine the path to list
	basePath := ws.StoragePath
	subPath := ""
	if len(args) > 1 {
		subPath = args[1]
	}

	// Construct full path
	fullPath := filepath.Join(basePath, subPath)

	// Ensure the path is within the storage directory
	cleanPath := filepath.Clean(fullPath)
	cleanBasePath := filepath.Clean(basePath)
	if !strings.HasPrefix(cleanPath, cleanBasePath) {
		return fmt.Errorf("path traversal attempt detected")
	}

	// Check if the path exists
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			ui.Warning("Path does not exist: %s", subPath)
			return nil
		}
		return fmt.Errorf("failed to stat path: %w", err)
	}

	// If it's a file, show file info
	if !info.IsDir() {
		ui.Success("File: %s", subPath)
		ui.Output("Size: %d bytes", info.Size())
		ui.Output("Modified: %s", info.ModTime().Format("2006-01-02 15:04:05"))
		return nil
	}

	// List directory contents
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if len(entries) == 0 {
		ui.Info("Directory is empty")
		return nil
	}

	// Display the listing
	if subPath != "" {
		ui.PrintSectionHeader("", fmt.Sprintf("Contents of %s", subPath), len(entries))
	} else {
		ui.PrintSectionHeader("", "Storage contents", len(entries))
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			ui.Output("%s/", name)
		} else {
			info, _ := entry.Info()
			ui.Output("%s (%d bytes)", name, info.Size())
		}
	}

	ui.Info("Total: %d items", len(entries))

	return nil
}
