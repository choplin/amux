// Package hooks provides CLI commands for managing amux hooks.
package hooks

import (
	"github.com/spf13/cobra"
)

// Cmd is the main hooks command.
var Cmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage workspace lifecycle hooks",
	Long: `Manage hooks that run automatically during workspace lifecycle events.

Hooks allow you to automate tasks like installing dependencies, setting up
development environments, or preparing context for AI agents.`,
}

func init() {
	Cmd.AddCommand(hooksInitCmd)
	Cmd.AddCommand(hooksTrustCmd)
	Cmd.AddCommand(hooksListCmd)
	Cmd.AddCommand(hooksTestCmd)
}
