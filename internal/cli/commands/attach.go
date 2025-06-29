package commands

import (
	"github.com/aki/amux/internal/cli/commands/session"
	"github.com/spf13/cobra"
)

// NewAttachCommand creates a shortcut for session attach
func NewAttachCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "attach <session-id>",
		Short: "Attach to a running session (shortcut for 'session attach')",
		Long: `Attach to a running session.

This is a shortcut for 'amux session attach'.

This is only supported for tmux runtime sessions.`,
		Args: cobra.ExactArgs(1),
		RunE: session.AttachSession,
	}
}
