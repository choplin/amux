package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/storage"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <session-id> <path>",
	Short: "Read a file from session storage",
	Long:  "Read and display the contents of a file from session storage",
	Args:  cobra.ExactArgs(2),
	RunE:  runRead,
}

func runRead(cmd *cobra.Command, args []string) error {
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

	// Get session info
	info := sess.Info()
	if info.StoragePath == "" {
		return fmt.Errorf("storage path not found for session")
	}

	// Create storage manager
	storageManager := storage.NewManager()

	// Read the file
	path := args[1]
	content, err := storageManager.ReadFile(ctx, info.StoragePath, path)
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
