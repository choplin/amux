// Package config provides CLI command implementations for amux config management.
package config

import (
	"github.com/spf13/cobra"
)

var showFormat string

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage amux configuration",
	Long: `Manage amux configuration including agent definitions, project settings, and other configuration options.

The config command provides subcommands to view and edit the configuration file.`,
	Example: `  # View current configuration
  amux config show

  # Edit configuration in your editor
  amux config edit

  # Validate configuration
  amux config validate`,
}

func init() {
	// Add subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configValidateCmd())

	// Configure flags
	configShowCmd.Flags().StringVar(&showFormat, "format", "yaml", "Output format (yaml, json, pretty)")
}

// Command returns the config command
func Command() *cobra.Command {
	return configCmd
}
