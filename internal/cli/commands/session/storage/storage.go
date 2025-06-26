// Package storage provides CLI commands for managing session storage.
package storage

import (
	"github.com/spf13/cobra"
)

var storageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Manage session storage",
	Long:  "Browse, read, write, and remove files in session storage",
}

func init() {
	// Add subcommands
	storageCmd.AddCommand(listCmd)
	storageCmd.AddCommand(readCmd)
	storageCmd.AddCommand(writeCmd)
	storageCmd.AddCommand(removeCmd)
}

// Command returns the storage command
func Command() *cobra.Command {
	return storageCmd
}
