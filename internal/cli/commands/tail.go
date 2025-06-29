package commands

import (
	"github.com/aki/amux/internal/cli/commands/session"
	"github.com/spf13/cobra"
)

// NewTailCommand creates a shortcut for session logs --follow
func NewTailCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tail <session-id>",
		Short: "Follow logs from a session (shortcut for 'session logs --follow')",
		Long: `Follow logs from a session in real-time.

This is a shortcut for 'amux session logs --follow'.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Force follow mode for tail command
			session.BindLogsFlags(cmd)
			session.SetLogsFollow(true)
			return session.ShowLogs(cmd, args)
		},
	}

	// Add tail-specific flags
	cmd.Flags().IntP("tail", "n", 0, "Number of lines to show from the end")

	return cmd
}
