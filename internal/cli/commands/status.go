package commands

import (
	"github.com/aki/amux/internal/cli/commands/session"
	"github.com/spf13/cobra"
)

// NewStatusCommand creates a shortcut for session list --format=wide
func NewStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show detailed status of all sessions (shortcut for 'session list --format=wide')",
		Long: `Show detailed status of all sessions across all workspaces.

This is a shortcut for 'amux session list --all --format=wide'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Force wide format and all workspaces for status command
			session.BindListFlags(cmd)
			session.SetListAll(true)
			session.SetListFormat("wide")
			return session.ListSessions(cmd, args)
		},
	}

	return cmd
}
