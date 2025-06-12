package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/agent"
	"github.com/aki/amux/internal/core/config"
)

var (
	// Flags for agent config commands
	agentName    string
	agentType    string
	agentCommand string
	agentEnv     []string
)

var agentConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage agent configurations",
	Long:  `Manage AI agent configurations including commands and environment variables.`,
}

var agentConfigListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List configured agents",
	RunE:    listAgentConfigs,
}

var agentConfigAddCmd = &cobra.Command{
	Use:   "add <id>",
	Short: "Add a new agent configuration",
	Long: `Add a new agent configuration.

Examples:
  # Add a simple agent
  amux agent config add gpt --name "GPT-4" --type openai

  # Add an agent with custom command
  amux agent config add claude-opus --name "Claude Opus" --command "claude --model opus"

  # Add an agent with environment variables
  amux agent config add gpt --name "GPT-4" --env OPENAI_API_KEY=sk-...`,
	Args: cobra.ExactArgs(1),
	RunE: addAgentConfig,
}

var agentConfigUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an agent configuration",
	Long: `Update an existing agent configuration.

Examples:
  # Update agent command
  amux agent config update claude --command "claude --model sonnet"

  # Update environment variables
  amux agent config update claude --env ANTHROPIC_API_KEY=sk-new...`,
	Args: cobra.ExactArgs(1),
	RunE: updateAgentConfig,
}

var agentConfigRemoveCmd = &cobra.Command{
	Use:     "remove <id>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove an agent configuration",
	Args:    cobra.ExactArgs(1),
	RunE:    removeAgentConfig,
}

var agentConfigShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show detailed agent configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  showAgentConfig,
}

func init() {
	// Add subcommands
	agentConfigCmd.AddCommand(agentConfigListCmd)
	agentConfigCmd.AddCommand(agentConfigAddCmd)
	agentConfigCmd.AddCommand(agentConfigUpdateCmd)
	agentConfigCmd.AddCommand(agentConfigRemoveCmd)
	agentConfigCmd.AddCommand(agentConfigShowCmd)
	agentCmd.AddCommand(agentConfigCmd)

	// Add flags
	agentConfigAddCmd.Flags().StringVar(&agentName, "name", "", "Agent display name")
	agentConfigAddCmd.Flags().StringVar(&agentType, "type", "", "Agent type")
	agentConfigAddCmd.Flags().StringVar(&agentCommand, "command", "", "Command to run the agent")
	agentConfigAddCmd.Flags().StringSliceVar(&agentEnv, "env", []string{}, "Environment variables (KEY=VALUE)")

	agentConfigUpdateCmd.Flags().StringVar(&agentName, "name", "", "Agent display name")
	agentConfigUpdateCmd.Flags().StringVar(&agentType, "type", "", "Agent type")
	agentConfigUpdateCmd.Flags().StringVar(&agentCommand, "command", "", "Command to run the agent")
	agentConfigUpdateCmd.Flags().StringSliceVar(&agentEnv, "env", []string{}, "Environment variables (KEY=VALUE)")
}

func listAgentConfigs(cmd *cobra.Command, args []string) error {
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
		return nil
	}

	// Print header
	ui.OutputLine("%-15s %-20s %-15s %-30s", "ID", "NAME", "TYPE", "COMMAND")
	ui.Separator("-", 80)

	// Print agents
	for id, agent := range agents {
		command := agent.Command
		if command == "" {
			command = ui.DimStyle.Render("(default: " + id + ")")
		}
		ui.OutputLine("%-15s %-20s %-15s %-30s", id, agent.Name, agent.Type, command)
	}

	return nil
}

func addAgentConfig(cmd *cobra.Command, args []string) error {
	agentID := args[0]

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	// Create managers
	configManager := config.NewManager(projectRoot)
	agentManager := agent.NewManager(configManager)

	// Check if agent already exists
	if _, err := agentManager.GetAgent(agentID); err == nil {
		return fmt.Errorf("agent '%s' already exists", agentID)
	}

	// Parse environment variables
	env := make(map[string]string)
	for _, envVar := range agentEnv {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid environment variable format: %s (expected KEY=VALUE)", envVar)
		}
		env[parts[0]] = parts[1]
	}

	// Create agent
	newAgent := config.Agent{
		Name:        agentName,
		Type:        agentType,
		Command:     agentCommand,
		Environment: env,
	}

	// Set defaults
	if newAgent.Name == "" {
		newAgent.Name = agentID
	}
	if newAgent.Type == "" {
		newAgent.Type = "custom"
	}

	// Add agent
	if err := agentManager.AddAgent(agentID, newAgent); err != nil {
		return fmt.Errorf("failed to add agent: %w", err)
	}

	ui.Success("Added agent '%s'", agentID)
	return nil
}

func updateAgentConfig(cmd *cobra.Command, args []string) error {
	agentID := args[0]

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	// Create managers
	configManager := config.NewManager(projectRoot)
	agentManager := agent.NewManager(configManager)

	// Get existing agent
	existingAgent, err := agentManager.GetAgent(agentID)
	if err != nil {
		return fmt.Errorf("agent '%s' not found", agentID)
	}

	// Update fields
	if cmd.Flags().Changed("name") {
		existingAgent.Name = agentName
	}
	if cmd.Flags().Changed("type") {
		existingAgent.Type = agentType
	}
	if cmd.Flags().Changed("command") {
		existingAgent.Command = agentCommand
	}
	if cmd.Flags().Changed("env") {
		// Parse environment variables
		env := make(map[string]string)
		for _, envVar := range agentEnv {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid environment variable format: %s (expected KEY=VALUE)", envVar)
			}
			env[parts[0]] = parts[1]
		}
		existingAgent.Environment = env
	}

	// Update agent
	if err := agentManager.UpdateAgent(agentID, *existingAgent); err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	ui.Success("Updated agent '%s'", agentID)
	return nil
}

func removeAgentConfig(cmd *cobra.Command, args []string) error {
	agentID := args[0]

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	// Create managers
	configManager := config.NewManager(projectRoot)
	agentManager := agent.NewManager(configManager)

	// Remove agent
	if err := agentManager.RemoveAgent(agentID); err != nil {
		return fmt.Errorf("failed to remove agent: %w", err)
	}

	ui.Success("Removed agent '%s'", agentID)
	return nil
}

func showAgentConfig(cmd *cobra.Command, args []string) error {
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

	if agent.Command != "" {
		ui.PrintIndented(2, "Command: %s", agent.Command)
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

	return nil
}
