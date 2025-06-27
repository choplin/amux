// Package session implements session-related CLI commands.
package session

import (
	"github.com/aki/amux/internal/cli/commands/session/storage"
	"github.com/spf13/cobra"
)

var (
	// Flags for session run
	runWorkspace          string
	runCommand            string
	runEnv                []string
	runInitialPrompt      string
	runName               string
	runDescription        string
	runSessionName        string
	runSessionDescription string
	runNoHooks            bool

	// Flags for session logs
	followLogs     bool
	followInterval string
)

// Command returns the session command
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage agent sessions",
		Long: `Manage agent sessions running in multiplexed workspaces.

Sessions are running instances of agents. You can run multiple sessions
concurrently in isolated workspaces, attach to running sessions, and
manage their lifecycle.`,
	}

	// Add subcommands
	cmd.AddCommand(runCmd())
	cmd.AddCommand(listCmd())
	cmd.AddCommand(attachCmd())
	cmd.AddCommand(stopCmd())
	cmd.AddCommand(logsCmd())
	cmd.AddCommand(removeCmd())
	cmd.AddCommand(sendInputCmd())
	cmd.AddCommand(storage.Command())

	return cmd
}

// TailCommand returns the tail command (alias for session logs -f)
func TailCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tail <session>",
		Short: "Follow agent session logs in real-time",
		Long:  "Continuously stream output from an agent session. Similar to 'session logs -f'.",
		Args:  cobra.ExactArgs(1),
		RunE:  tailSessionLogs,
	}
}
