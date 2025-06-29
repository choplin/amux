package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/aki/amux/internal/cli/commands/config"
	"github.com/aki/amux/internal/cli/commands/hooks"
	"github.com/aki/amux/internal/cli/commands/session"
	"github.com/aki/amux/internal/cli/commands/workspace"
	"github.com/aki/amux/internal/cli/ui"
	runtimeinit "github.com/aki/amux/internal/runtime/init"
)

var formatFlag string

var rootCmd = &cobra.Command{
	Use: "amux",

	Short: "Agent Multiplexer - Orchestrate AI agents in isolated workspaces",

	Long: `Amux (Agent Multiplexer) provides isolated git worktree-based environments where AI agents

can work independently without context mixing. It enables multiplexing multiple agent sessions
across different workspaces.`,

	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize slog logger first
		InitializeSlog()

		// Parse and set the global formatter
		format, err := ui.ParseFormat(formatFlag)
		if err != nil {
			return err
		}
		return ui.SetGlobalFormatter(format)
	},
}

func init() {
	// Initialize runtime registry
	_ = runtimeinit.RegisterDefaults()
	// Ignore error - runtime can be initialized later
	// This is important for tests and some commands that don't need runtime

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&formatFlag, "format", "pretty", "Output format (pretty, json)")

	// Register global logger flags
	RegisterLoggerFlags(rootCmd)

	// Add subcommands

	rootCmd.AddCommand(initCmd)

	rootCmd.AddCommand(workspace.Command())

	rootCmd.AddCommand(session.Command())

	rootCmd.AddCommand(mcpCmd)

	rootCmd.AddCommand(config.Command())

	// Add global aliases for common session operations
	// These are shortcuts to session subcommands

	// Get the session commands to create aliases
	sessionCmd := session.Command()

	// Find the subcommands to create aliases
	var runSubCmd, listSubCmd, attachSubCmd *cobra.Command
	for _, cmd := range sessionCmd.Commands() {
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
			Short: "Alias for 'session run'",
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
			Short: "Alias for 'session list'",
			RunE:  listSubCmd.RunE,
		}
		rootCmd.AddCommand(psCmd)

		// Add status alias too (same as ps/list)
		statusCmd := &cobra.Command{
			Use:   "status",
			Short: "Show status of agent sessions (alias for 'session list')",
			RunE:  listSubCmd.RunE,
		}
		rootCmd.AddCommand(statusCmd)
	}

	if attachSubCmd != nil {
		attachCmd := &cobra.Command{
			Use:   "attach <session>",
			Short: "Alias for 'session attach'",
			Args:  attachSubCmd.Args,
			RunE:  attachSubCmd.RunE,
		}
		rootCmd.AddCommand(attachCmd)
	}

	// Add tail command alias
	rootCmd.AddCommand(session.TailCommand())

	// Add hooks command
	rootCmd.AddCommand(hooks.Cmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
