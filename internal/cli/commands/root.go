package commands

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{

	Use: "amux",

	Short: "Agent Multiplexer - Orchestrate AI agents in isolated workspaces",

	Long: `Amux (Agent Multiplexer) provides isolated git worktree-based environments where AI agents

can work independently without context mixing. It enables multiplexing multiple agent sessions
across different workspaces.`,
}

func init() {

	// Add subcommands

	rootCmd.AddCommand(initCmd)

	rootCmd.AddCommand(workspaceCmd)

	rootCmd.AddCommand(agentCmd)

	rootCmd.AddCommand(mcpCmd)

	// Add global aliases for common agent operations
	// These are shortcuts to agent subcommands

	runCmd := &cobra.Command{
		Use:   "run <agent>",
		Short: "Alias for 'agent run'",
		Long:  agentRunCmd.Long,
		Args:  cobra.ExactArgs(1),
		RunE:  runAgent,
	}
	// Copy flags from agent run command
	runCmd.Flags().StringVarP(&runWorkspace, "workspace", "w", "", "Workspace to run agent in (name or ID)")
	runCmd.Flags().StringVarP(&runCommand, "command", "c", "", "Override agent command")
	runCmd.Flags().StringSliceVarP(&runEnv, "env", "e", []string{}, "Environment variables (KEY=VALUE)")

	psCmd := &cobra.Command{
		Use:   "ps",
		Short: "Alias for 'agent list'",
		RunE:  listAgents,
	}

	attachCmd := &cobra.Command{
		Use:   "attach <session>",
		Short: "Alias for 'agent attach'",
		Args:  cobra.ExactArgs(1),
		RunE:  attachAgent,
	}

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(psCmd)
	rootCmd.AddCommand(attachCmd)

}

// Execute runs the root command

func Execute() error {

	return rootCmd.Execute()

}
