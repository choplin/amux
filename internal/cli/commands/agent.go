package commands

import (
	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage AI agent sessions (future functionality)",
	Long: `Manage AI agent sessions in multiplexed workspaces.

This functionality is planned for future releases and will enable:
- Running multiple AI agents concurrently
- Attaching to running agent sessions
- Managing agent lifecycle`,
}

var agentRunCmd = &cobra.Command{
	Use:   "run <agent> [options]",
	Short: "Run an AI agent in a workspace (not yet implemented)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Info("Agent multiplexing functionality is not yet implemented")
		ui.Info("This will allow running AI agents in isolated workspaces")
		return nil
	},
}

var agentListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List running agent sessions (not yet implemented)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Info("Agent multiplexing functionality is not yet implemented")
		ui.Info("This will show all running agent sessions")
		return nil
	},
}

var agentAttachCmd = &cobra.Command{
	Use:   "attach <session>",
	Short: "Attach to a running agent session (not yet implemented)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Info("Agent multiplexing functionality is not yet implemented")
		ui.Info("This will allow attaching to running agent sessions")
		return nil
	},
}

var agentStopCmd = &cobra.Command{
	Use:   "stop <session>",
	Short: "Stop a running agent session (not yet implemented)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Info("Agent multiplexing functionality is not yet implemented")
		ui.Info("This will stop a running agent session")
		return nil
	},
}

func init() {
	// Add subcommands
	agentCmd.AddCommand(agentRunCmd)
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentAttachCmd)
	agentCmd.AddCommand(agentStopCmd)

}
