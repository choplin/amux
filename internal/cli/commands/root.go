package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/aki/amux/internal/cli/commands/agent"
	"github.com/aki/amux/internal/cli/ui"
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

	rootCmd.AddCommand(agent.Command())

	rootCmd.AddCommand(mcpCmd)

	// Add global aliases for common agent operations
	// These are shortcuts to agent subcommands

	// Get the agent commands to create aliases
	agentCmd := agent.Command()

	// Find the subcommands to create aliases
	var runSubCmd, listSubCmd, attachSubCmd *cobra.Command
	for _, cmd := range agentCmd.Commands() {
		switch cmd.Use {
		case "run <agent>":
			runSubCmd = cmd
		case "list":
			listSubCmd = cmd
		case "attach <session>":
			attachSubCmd = cmd
		}
	}

	// Create alias commands
	if runSubCmd != nil {
		runCmd := &cobra.Command{
			Use:   "run <agent>",
			Short: "Alias for 'agent run'",
			Long:  runSubCmd.Long,
			Args:  runSubCmd.Args,
			RunE:  runSubCmd.RunE,
		}
		// Copy flags
		runSubCmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			runCmd.Flags().AddFlag(f)
		})
		rootCmd.AddCommand(runCmd)
	}

	if listSubCmd != nil {
		psCmd := &cobra.Command{
			Use:   "ps",
			Short: "Alias for 'agent list'",
			RunE:  listSubCmd.RunE,
		}
		rootCmd.AddCommand(psCmd)
	}

	if attachSubCmd != nil {
		attachCmd := &cobra.Command{
			Use:   "attach <session>",
			Short: "Alias for 'agent attach'",
			Args:  attachSubCmd.Args,
			RunE:  attachSubCmd.RunE,
		}
		rootCmd.AddCommand(attachCmd)
	}

	// Add tail command alias
	rootCmd.AddCommand(agent.TailCommand())

	// Add mailbox command
	rootCmd.AddCommand(mailboxCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
