package config

import (
	"fmt"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display the current configuration",
	Long:  "Display the current amux configuration in a readable format",
	Example: `  # Show configuration in YAML format (default)
  amux config show

  # Show configuration in JSON format
  amux config show --format json

  # Show configuration in pretty format
  amux config show --format pretty`,
	RunE: runConfigShow,
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	mgr := config.NewManager(projectRoot)
	cfg, err := mgr.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	switch showFormat {
	case "json":
		return ui.GlobalFormatter.Output(cfg)
	case "yaml":
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal configuration: %w", err)
		}
		ui.Output("%s", string(data))
		return nil
	case "pretty":
		return showConfigPretty(cfg)
	default:
		return fmt.Errorf("unsupported format: %s", showFormat)
	}
}

func showConfigPretty(cfg *config.Config) error {
	ui.OutputLine("MCP Configuration:")
	ui.OutputLine("  Transport: %s", cfg.MCP.Transport.Type)
	if cfg.MCP.Transport.Type == "http" && cfg.MCP.Transport.HTTP.Port > 0 {
		ui.OutputLine("  Port: %d", cfg.MCP.Transport.HTTP.Port)
	}

	if len(cfg.Agents) > 0 {
		ui.OutputLine("\nAgent Definitions:")
		for id, agent := range cfg.Agents {
			ui.OutputLine("  %s:", id)
			if agent.Name != "" {
				ui.OutputLine("    Name: %s", agent.Name)
			}
			ui.OutputLine("    Runtime: %s", agent.Runtime)
			if len(agent.Command) > 0 {
				ui.OutputLine("    Command: %v", agent.Command)
			}
			if agent.WorkingDir != "" {
				ui.OutputLine("    Working Directory: %s", agent.WorkingDir)
			}
			if len(agent.Environment) > 0 {
				ui.OutputLine("    Environment:")
				for k, v := range agent.Environment {
					ui.OutputLine("      %s: %s", k, v)
				}
			}
		}
	} else {
		ui.OutputLine("\nNo agents configured.")
	}

	return nil
}
