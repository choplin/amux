package commands

import (
	"github.com/aki/amux/internal/cli/ui"
	"github.com/spf13/cobra"
)

var formatFlag string

var rootCmd = &cobra.Command{
	Use: "amux",

	Short: "Agent Multiplexer - Orchestrate AI agents in isolated workspaces",

	Long: `Amux (Agent Multiplexer) provides isolated git worktree-based environments where AI agents

can work independently without context mixing. It enables multiplexing multiple agent sessions
across different workspaces.`,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Parse and set the global formatter
		format, err := ui.ParseFormat(formatFlag)
		if err != nil {
			return err
		}
		return ui.SetGlobalFormatter(format)
	},
}

func init() {
	// Add global flags
	rootCmd.PersistentFlags().StringVar(&formatFlag, "format", "pretty", "Output format (pretty, json)")

	// Register global logger flags
	RegisterLoggerFlags(rootCmd)

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

	tailCmd := &cobra.Command{
		Use:   "tail <session>",
		Short: "Follow agent session logs in real-time",
		Long:  "Continuously stream output from an agent session. Similar to 'agent logs -f'.",
		Args:  cobra.ExactArgs(1),
		RunE:  tailAgentLogs,
	}

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(psCmd)
	rootCmd.AddCommand(attachCmd)
	rootCmd.AddCommand(tailCmd)

	// Add mailbox command
	rootCmd.AddCommand(mailboxCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
