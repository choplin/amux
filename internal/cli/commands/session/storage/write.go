package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/config"
	"github.com/spf13/cobra"
)

var writeCmd = &cobra.Command{
	Use:   "write <session-id> <filename>",
	Short: "Write a file to session storage",
	Long: `Write a file to a session's storage directory.

The content is read from stdin.

Example:
  echo "Hello, world!" | amux session storage write 1 hello.txt`,
	Args: cobra.ExactArgs(2),
	RunE: writeStorage,
}

func writeStorage(cmd *cobra.Command, args []string) error {
	sessionID := args[0]
	filename := args[1]

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create config manager
	configMgr := config.NewManager(wd)
	if !configMgr.IsInitialized() {
		return fmt.Errorf("amux not initialized. Run 'amux init' first")
	}

	// Get storage directory
	storageDir := filepath.Join(configMgr.GetAmuxDir(), "sessions", fmt.Sprintf("session-%s", sessionID), "storage")

	// Create storage directory if needed
	if err := os.MkdirAll(storageDir, 0o755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Get file path
	filePath := filepath.Join(storageDir, filename)

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy from stdin
	n, err := io.Copy(file, os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	ui.Success("Wrote %d bytes to %s", n, filename)
	return nil
}
