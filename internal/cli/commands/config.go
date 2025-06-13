package commands

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage amux configuration",
	Long: `Manage amux configuration including agent definitions, project settings, and other configuration options.

The config command provides subcommands to view and edit the configuration file.`,
	Example: `  # View current configuration
  amux config show

  # Edit configuration in your editor
  amux config edit`,
}

func init() {
	rootCmd.AddCommand(configCmd)
}
