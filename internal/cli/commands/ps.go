package commands

import (
	"github.com/aki/amux/internal/cli/commands/session"
	"github.com/spf13/cobra"
)

// NewPsCommand creates a shortcut for session list
func NewPsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List running sessions (shortcut for 'session list')",
		Long: `List running sessions.

This is a shortcut for 'amux session list'.

By default, shows only sessions in the current workspace.
Use --all to show sessions from all workspaces.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bind flags to session.listOpts
			session.BindListFlags(cmd)
			return session.ListSessions(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().StringP("workspace", "w", "", "Filter by workspace")
	cmd.Flags().BoolP("all", "a", false, "Show sessions from all workspaces")
	cmd.Flags().StringP("format", "f", "", "Output format (json, wide)")

	return cmd
}
