package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
)

func init() {
	configCmd.AddCommand(configValidateCmd())
}

func configValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long: `Validate the configuration file to ensure it conforms to the expected format.

This command checks:
- Required fields are present
- Field types are correct
- Agent configurations are valid
- Session types are supported`,
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

	// Load configuration (this will validate it)
	cfg, err := configManager.Load()
	if err != nil {
		ui.Error("Configuration validation failed: %v", err)
		return fmt.Errorf("invalid configuration")
	}

	// If we get here, the configuration is valid
	ui.Success("Configuration is valid")

	// Show configuration details if verbose
	if verbose {
		ui.Info("")
		ui.Info("Configuration details:")
		ui.Info("  Version: %s", cfg.Version)
		ui.Info("  Project: %s", cfg.Project.Name)

		if cfg.Project.Repository != "" {
			ui.Info("  Repository: %s", cfg.Project.Repository)
		}

		if cfg.Project.DefaultAgent != "" {
			ui.Info("  Default Agent: %s", cfg.Project.DefaultAgent)
		}

		ui.Info("")
		ui.Info("MCP Configuration:")
		ui.Info("  Transport: %s", cfg.MCP.Transport.Type)

		if len(cfg.Agents) > 0 {
			ui.Info("")
			ui.Info("Agents (%d configured):", len(cfg.Agents))
			for id, agent := range cfg.Agents {
				ui.Info("  %s:", id)
				ui.Info("    Name: %s", agent.Name)
				ui.Info("    Type: %s", agent.Type)

				switch agent.Type {
				case "tmux":
					if agent.Tmux != nil {
						ui.Info("    Command: %s", agent.Tmux.Command)
						if agent.Tmux.Shell != "" {
							ui.Info("    Shell: %s", agent.Tmux.Shell)
						}
						if agent.Tmux.WindowName != "" {
							ui.Info("    Window Name: %s", agent.Tmux.WindowName)
						}
					}
				}

				if agent.Description != "" {
					ui.Info("    Description: %s", agent.Description)
				}

				if len(agent.Environment) > 0 {
					ui.Info("    Environment Variables: %d", len(agent.Environment))
				}
			}
		} else {
			ui.Info("")
			ui.Info("No agents configured")
		}
	}

	return nil
}
