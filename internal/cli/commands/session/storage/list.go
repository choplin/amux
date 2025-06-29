// Package storage provides storage management commands for sessions
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list <session-id>",
	Short: "List files in session storage",
	Long:  `List all files stored in a session's storage directory.`,
	Args:  cobra.ExactArgs(1),
	RunE:  listStorage,
}

func listStorage(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

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

	// Check if storage directory exists
	if _, err := os.Stat(storageDir); os.IsNotExist(err) {
		ui.Info("No storage found for session: %s", sessionID)
		return nil
	}

	// List files
	entries, err := os.ReadDir(storageDir)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}

	if len(entries) == 0 {
		ui.Info("Storage is empty")
		return nil
	}

	// Display files
	fmt.Println("Files in session storage:")
	fmt.Println(strings.Repeat("-", 40))
	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Printf("%s/\n", entry.Name())
		} else {
			info, _ := entry.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			fmt.Printf("%s (%d bytes)\n", entry.Name(), size)
		}
	}

	return nil
}
