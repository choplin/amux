package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aki/amux/internal/core/session"
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
	manager, err := getSessionManager()
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
