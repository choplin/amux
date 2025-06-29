package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
)

func configValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long: `Validate the configuration file to ensure it conforms to the expected format.

This command validates the configuration file using JSON Schema to ensure:
- Required fields are present
- Field types match the schema
- Agent configurations are valid
- Session types are supported
- Additional properties are not allowed`,
		Example: `  # Validate current project configuration
  amux config validate

  # Validate with verbose output
  amux config validate --verbose`,
		RunE: validateConfig,
	}

	cmd.Flags().BoolP("verbose", "v", false, "Show detailed validation information")

	return cmd
}

func validateConfig(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	// Create config manager
	configManager := config.NewManager(projectRoot)

	// Load and validate configuration using JSON Schema
	// The Load() method internally uses LoadWithValidation() which:
	// 1. Validates YAML against JSON Schema
	// 2. Unmarshals to Config struct
	// 3. Applies defaults
	cfg, err := configManager.Load()
	if err != nil {
		ui.Error("Configuration validation failed: %v", err)
		return fmt.Errorf("invalid configuration")
	}

	// If we get here, the configuration is valid
	ui.Success("Configuration is valid")

	// Show configuration details if verbose
	if verbose {
		ui.OutputLine("")
		ui.OutputLine("Configuration details:")
		ui.OutputLine("  Version: %s", cfg.Version)

		ui.OutputLine("")
		ui.OutputLine("MCP Configuration:")
		ui.OutputLine("  Transport: %s", cfg.MCP.Transport.Type)

		if len(cfg.Agents) > 0 {
			ui.OutputLine("")
			ui.OutputLine("Agents (%d configured):", len(cfg.Agents))
			for id, agent := range cfg.Agents {
				ui.OutputLine("  %s:", id)
				ui.OutputLine("    Name: %s", agent.Name)
				ui.OutputLine("    Runtime: %s", agent.Runtime)
				if len(agent.Command) > 0 {
					ui.OutputLine("    Command: %v", agent.Command)
				}

				if agent.Description != "" {
					ui.OutputLine("    Description: %s", agent.Description)
				}

				if len(agent.Environment) > 0 {
					ui.OutputLine("    Environment Variables: %d", len(agent.Environment))
				}
			}
		} else {
			ui.OutputLine("")
			ui.OutputLine("No agents configured")
		}
	}

	return nil
}
