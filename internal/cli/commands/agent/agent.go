// Package agent implements agent configuration CLI commands.
package agent

import (
	"github.com/spf13/cobra"
)

// Command returns the agent command
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "View agent configurations",
		Long: `View configured AI agents.

Agents are static configurations that define which AI tool to run,
default commands, and environment variables. To modify agent configurations,
use 'amux config edit'.`,
	}

	// Add subcommands
	cmd.AddCommand(listCmd())
	cmd.AddCommand(showCmd())

	return cmd
}
