package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aki/amux/internal/config"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <session-id> <filename>",
	Short: "Read a file from session storage",
	Long:  `Read and display a file from a session's storage directory.`,
	Args:  cobra.ExactArgs(2),
	RunE:  readStorage,
}

func readStorage(cmd *cobra.Command, args []string) error {
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

	// Read file
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filename)
		}
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	// Copy to stdout
	if _, err := io.Copy(os.Stdout, file); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	return nil
}
