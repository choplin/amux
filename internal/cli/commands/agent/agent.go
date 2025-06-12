// Package agent implements agent-related CLI commands.
package agent

import (
	"github.com/spf13/cobra"
)

var (
	// Flags for agent run
	runWorkspace string
	runCommand   string
	runEnv       []string

	// Flags for agent logs
	followLogs bool
)

// Command returns the agent command
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage AI agent sessions",
		Long: `Manage AI agent sessions in multiplexed workspaces.

Run multiple AI agents concurrently in isolated workspaces,
attach to running sessions, and manage agent lifecycle.`,
	}

	// Initialize config command
	initAgentConfig()

	// Add subcommands
	cmd.AddCommand(runCmd())
	cmd.AddCommand(listCmd())
	cmd.AddCommand(attachCmd())
	cmd.AddCommand(stopCmd())
	cmd.AddCommand(logsCmd())
	cmd.AddCommand(agentConfigCmd)

	return cmd
}

// TailCommand returns the tail command (alias for agent logs -f)
func TailCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tail <session>",
		Short: "Follow agent session logs in real-time",
		Long:  "Continuously stream output from an agent session. Similar to 'agent logs -f'.",
		Args:  cobra.ExactArgs(1),
		RunE:  tailAgentLogs,
	}
}
