package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/config"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <session-id> <filename>",
	Short: "Remove a file from session storage",
	Long:  `Remove a file from a session's storage directory.`,
	Args:  cobra.ExactArgs(2),
	RunE:  removeStorage,
}

func removeStorage(cmd *cobra.Command, args []string) error {
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

	// Get file path
	filePath := filepath.Join(configMgr.GetAmuxDir(), "sessions", fmt.Sprintf("session-%s", sessionID), "storage", filename)

	// Remove file
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filename)
		}
		return fmt.Errorf("failed to remove file: %w", err)
	}

	ui.Success("Removed %s", filename)
	return nil
}
