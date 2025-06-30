package storage

import (
	"github.com/spf13/cobra"
)

// Command returns the session storage command
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "storage",
		Aliases: []string{"st"},
		Short:   "Manage session storage",
		Long:    `Manage storage associated with sessions.`,
	}

	// Add subcommands
	cmd.AddCommand(listCmd)
	cmd.AddCommand(readCmd)
	cmd.AddCommand(writeCmd)
	cmd.AddCommand(removeCmd)

	return cmd
}
