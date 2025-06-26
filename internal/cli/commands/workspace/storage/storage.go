// Package storage provides CLI commands for managing workspace storage.
package storage

import (
	"github.com/spf13/cobra"
)

var storageCmd = &cobra.Command{
	Use:   "storage",
	Short: "Manage workspace storage",
	Long:  "Browse, read, write, and remove files in workspace storage",
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
