package agent

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/agent"
	"github.com/aki/amux/internal/core/config"
)

func showCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <agent>",
		Short: "Show agent configuration details",
		Long: `Show detailed configuration for a specific agent.

Displays the agent's name, type, command, and environment variables.`,
		Args: cobra.ExactArgs(1),
		RunE: showAgent,
	}
}

func showAgent(cmd *cobra.Command, args []string) error {
	agentID := args[0]

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	// Create managers
	configManager := config.NewManager(projectRoot)
	agentManager := agent.NewManager(configManager)

	// Get agent
	agent, err := agentManager.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("agent '%s' not found", agentID)
	}

	// Display agent details
	ui.OutputLine("Agent: %s", ui.BoldStyle.Render(agentID))
	ui.PrintIndented(2, "Name:    %s", agent.Name)
	ui.PrintIndented(2, "Type:    %s", agent.Type)

	if agent.Tmux != nil && agent.Tmux.Command != "" {
		ui.PrintIndented(2, "Command: %s", agent.Tmux.Command)
	} else {
		ui.PrintIndented(2, "Command: %s", ui.DimStyle.Render("(default: "+agentID+")"))
	}

	if len(agent.Environment) > 0 {
		ui.PrintIndented(2, "Environment:")
		for k, v := range agent.Environment {
			// Mask sensitive values
			displayValue := v
			if strings.Contains(strings.ToLower(k), "key") || strings.Contains(strings.ToLower(k), "token") {
				if len(v) > 8 {
					displayValue = v[:4] + "..." + v[len(v)-4:]
				} else {
					displayValue = "***"
				}
			}
			ui.PrintIndented(4, "%s=%s", k, displayValue)
		}
	}

	ui.OutputLine("")
	ui.Info("To modify this agent, use 'amux config edit'")

	return nil
}
