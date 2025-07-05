package commands

import (
	"github.com/spf13/cobra"

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
	rootCmd.AddCommand(hooks.Cmd)

	// Add shortcut commands
	rootCmd.AddCommand(NewRunCommand())
	rootCmd.AddCommand(NewPsCommand())
	rootCmd.AddCommand(NewAttachCommand())
	rootCmd.AddCommand(NewStatusCommand())

	// Add internal commands
	rootCmd.AddCommand(NewProxyCommand())
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
