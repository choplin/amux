package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/agent"
	"github.com/aki/amux/internal/core/config"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured agents",
		Long: `List all configured AI agents.

Shows agent ID, name, type, and command for each configured agent.`,
		RunE: listAgents,
	}
}

func listAgents(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	// Create managers
	configManager := config.NewManager(projectRoot)
	agentManager := agent.NewManager(configManager)

	// List agents
	agents, err := agentManager.ListAgents()
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	if len(agents) == 0 {
		ui.Info("No agents configured")
		ui.Info("Add agents by editing the configuration with 'amux config edit'")
		return nil
	}

	// Create table
	tbl := ui.NewTable("ID", "NAME", "TYPE", "COMMAND")

	// Add rows
	for id, agent := range agents {
		command := ""
		if agent.Type == config.AgentTypeTmux {
			if params, err := agent.GetTmuxParams(); err == nil && params.Command != "" {
				command = params.Command
			} else {
				command = ui.DimStyle.Render("(default: " + id + ")")
			}
		} else {
			command = ui.DimStyle.Render("(default: " + id + ")")
		}
		tbl.AddRow(id, agent.Name, string(agent.Type), command)
	}

	// Print with header
	ui.PrintSectionHeader("🤖", "Configured Agents", len(agents))
	tbl.Print()
	ui.OutputLine("")

	return nil
}
