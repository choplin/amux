package session

import (
	"github.com/aki/amux/internal/cli/commands/session/storage"
	"github.com/spf13/cobra"
)

// Command returns the session command
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "session",
		Aliases: []string{"s"},
		Short:   "Manage sessions",
		Long:    `Manage amux sessions for running tasks and commands.`,
	}

	// Add subcommands
	cmd.AddCommand(runCmd)
	cmd.AddCommand(listCmd)
	cmd.AddCommand(attachCmd)
	cmd.AddCommand(stopCmd)
	cmd.AddCommand(logsCmd)
	cmd.AddCommand(removeCmd)
	cmd.AddCommand(storage.Command())

	return cmd
}
