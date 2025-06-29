package commands

import (
	"github.com/aki/amux/internal/cli/commands/session"
	"github.com/spf13/cobra"
)

// NewRunCommand creates a shortcut for session run
func NewRunCommand() *cobra.Command {
	// Create a wrapper that binds to the same flags as session run
	cmd := &cobra.Command{
		Use:   "run [task-name] [-- command args...]",
		Short: "Run a task or command in a session (shortcut for 'session run')",
		Long: `Run a task or command in a session.

This is a shortcut for 'amux session run'.

Examples:
  # Run a predefined task
  amux run dev

  # Run a custom command
  amux run -- npm start

  # Run in a specific workspace
  amux run dev --workspace myworkspace

  # Run with tmux runtime
  amux run dev --runtime tmux`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Bind flags to session.runOpts
			session.BindRunFlags(cmd)
			return session.RunSession(cmd, args)
		},
	}

	// Add flags that will be bound to session.runOpts
	cmd.Flags().StringP("workspace", "w", "", "Workspace to run in")
	cmd.Flags().StringP("runtime", "r", "local", "Runtime to use (local, tmux)")
	cmd.Flags().StringArrayP("env", "e", nil, "Environment variables (KEY=VALUE)")
	cmd.Flags().StringP("dir", "d", "", "Working directory")
	cmd.Flags().BoolP("follow", "f", false, "Follow logs")

	return cmd
}
