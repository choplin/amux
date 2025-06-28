package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/storage"
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

	// Determine the path to list
	subPath := ""
	if len(args) > 1 {
		subPath = args[1]
	}

	// List files
	result, err := storageManager.ListFiles(ctx, subPath)
	if err != nil {
		var notFound storage.ErrNotFound
		if errors.As(err, &notFound) {
			ui.Warning("Path does not exist: %s", subPath)
			return nil
		}
		return err
	}

	// Handle empty directory
	if len(result.Files) == 0 {
		ui.Info("Directory is empty")
		return nil
	}

	// Check if listing a single file
	if result.IsTargetFile {
		ui.Success("File: %s", subPath)
		ui.Output("Size: %d bytes", result.Files[0].Size)
		ui.Output("Modified: %s", result.Files[0].ModTime.Format("2006-01-02 15:04:05"))
		return nil
	}

	// Display the listing
	if subPath != "" {
		ui.PrintSectionHeader("", fmt.Sprintf("Contents of %s", subPath), len(result.Files))
	} else {
		ui.PrintSectionHeader("", "Storage contents", len(result.Files))
	}

	// Show total items first
	ui.Output("Total: %d items\n\n", len(result.Files))

	// Create table for better formatting
	tbl := ui.NewTable("NAME", "TYPE", "SIZE")

	for _, file := range result.Files {
		var fileType string
		var displayName string

		if file.IsSymlink {
			displayName = file.Name
			if file.LinkTarget != "" {
				// Shorten the link target for display
				target := file.LinkTarget
				if len(target) > 50 {
					// Show only the last part of the path
					parts := strings.Split(target, "/")
					if len(parts) > 2 {
						target = "..." + strings.Join(parts[len(parts)-2:], "/")
					}
				}
				displayName = fmt.Sprintf("%s -> %s", file.Name, target)
			}
			if file.IsDir {
				fileType = "Symlink (dir)"
			} else {
				fileType = "Symlink"
			}
		} else {
			displayName = file.Name
			if file.IsDir {
				fileType = "Directory"
			} else {
				fileType = "File"
			}
		}

		if file.IsDir {
			tbl.AddRow(displayName, fileType, "-")
		} else {
			tbl.AddRow(displayName, fileType, ui.FormatSize(file.Size))
		}
	}

	tbl.Print()

	return nil
}
