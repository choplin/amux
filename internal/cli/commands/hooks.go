package commands

import (
	"fmt"
	"os"
	"os/user"
	"time"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/hooks"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage workspace lifecycle hooks",
	Long: `Manage hooks that run automatically during workspace lifecycle events.

Hooks allow you to automate tasks like installing dependencies, setting up
development environments, or preparing context for AI agents.`,
}

var hooksInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize hooks configuration",
	Long: `Initialize a hooks configuration file with example hooks.

This creates a .amux/hooks.yaml file with commented examples showing
how to configure hooks for different events.`,
	RunE: initHooks,
}

var hooksTrustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Trust hooks in this project",
	Long: `Trust the hooks configured in this project.

For security, hooks must be explicitly trusted before they will run.
This command shows you all configured hooks and asks for confirmation.`,
	RunE: trustHooks,
}

var hooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured hooks",
	Long:  `List all hooks configured in this project.`,
	RunE:  listHooks,
}

var hooksTestCmd = &cobra.Command{
	Use:   "test <event>",
	Short: "Test hooks for a specific event",
	Long: `Test hooks for a specific event without actually running them.

This shows what hooks would be executed and with what environment
variables, but doesn't actually run the commands.`,
	Args: cobra.ExactArgs(1),
	RunE: testHooks,
}

func init() {
	hooksCmd.AddCommand(hooksInitCmd)
	hooksCmd.AddCommand(hooksTrustCmd)
	hooksCmd.AddCommand(hooksListCmd)
	hooksCmd.AddCommand(hooksTestCmd)
}

func initHooks(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("not in an amux project: %w", err)
	}

	configManager := config.NewManager(projectRoot)
	configDir := configManager.GetAmuxDir()

	// Check if hooks.yaml already exists
	hooksPath := fmt.Sprintf("%s/%s", configDir, hooks.HooksConfigFile)
	if _, err := os.Stat(hooksPath); err == nil {
		return fmt.Errorf("hooks configuration already exists at %s", hooksPath)
	}

	// Create example configuration
	exampleConfig := &hooks.Config{
		Hooks: map[string][]hooks.Hook{
			string(hooks.EventWorkspaceCreate): {
				{
					Name:    "Install dependencies (example)",
					Command: "# npm install",
					Timeout: "5m",
					OnError: hooks.ErrorStrategyWarn,
				},
				{
					Name:   "Setup AI context (example)",
					Script: "# .amux/scripts/setup-context.sh",
					Env: map[string]string{
						"AI_MODEL": "claude-3",
					},
				},
			},
			string(hooks.EventWorkspaceRemove): {
				{
					Name:    "Cleanup (example)",
					Command: "# echo 'Workspace ${AMUX_WORKSPACE_NAME} is being removed'",
					OnError: hooks.ErrorStrategyIgnore,
				},
			},
		},
	}

	if err := hooks.SaveConfig(configDir, exampleConfig); err != nil {
		return fmt.Errorf("failed to save hooks configuration: %w", err)
	}

	ui.Success("Created hooks configuration at %s", hooksPath)
	ui.Info("Edit this file to configure your hooks, then run 'amux hooks trust' to enable them.")

	return nil
}

func trustHooks(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("not in an amux project: %w", err)
	}

	configManager := config.NewManager(projectRoot)
	configDir := configManager.GetAmuxDir()

	// Load current configuration
	hooksConfig, err := hooks.LoadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to load hooks configuration: %w", err)
	}

	// Check if any hooks are configured
	totalHooks := 0
	for _, hookList := range hooksConfig.Hooks {
		totalHooks += len(hookList)
	}

	if totalHooks == 0 {
		ui.Warning("No hooks are configured in %s/%s", configDir, hooks.HooksConfigFile)
		return nil
	}

	// Display hooks for review
	ui.OutputLine("Review the following hooks before trusting:")
	ui.OutputLine("")

	for event, hookList := range hooksConfig.Hooks {
		if len(hookList) == 0 {
			continue
		}

		ui.OutputLine("%s:", event)
		for i, hook := range hookList {
			cmdStr := hook.Command
			if cmdStr == "" {
				cmdStr = hook.Script
			}
			ui.OutputLine("  %d. \"%s\" - %s", i+1, hook.Name, cmdStr)
		}
		ui.OutputLine("")
	}

	// Ask for confirmation
	if !ui.Confirm("Trust these hooks?") {
		ui.OutputLine("Hooks not trusted.")
		return nil
	}

	// Calculate hash and save trust info
	hash, err := hooks.CalculateConfigHash(hooksConfig)
	if err != nil {
		return fmt.Errorf("failed to calculate configuration hash: %w", err)
	}

	currentUser, _ := user.Current()
	trustInfo := &hooks.TrustInfo{
		Hash:      hash,
		TrustedAt: time.Now(),
		TrustedBy: currentUser.Username,
	}

	if err := hooks.SaveTrustInfo(configDir, trustInfo); err != nil {
		return fmt.Errorf("failed to save trust information: %w", err)
	}

	ui.Success("Hooks trusted. They will now run automatically during workspace operations.")
	return nil
}

func listHooks(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("not in an amux project: %w", err)
	}

	configManager := config.NewManager(projectRoot)
	configDir := configManager.GetAmuxDir()

	// Load configuration
	hooksConfig, err := hooks.LoadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to load hooks configuration: %w", err)
	}

	// Check trust status
	trusted, err := hooks.IsTrusted(configDir, hooksConfig)
	if err != nil {
		return fmt.Errorf("failed to check trust status: %w", err)
	}

	// Display hooks
	ui.OutputLine("Hooks configuration (%s):", func() string {
		if trusted {
			return "trusted"
		}
		return "not trusted"
	}())
	ui.OutputLine("")

	hasHooks := false
	for event, hookList := range hooksConfig.Hooks {
		if len(hookList) == 0 {
			continue
		}

		hasHooks = true
		ui.OutputLine("%s:", event)
		for i, hook := range hookList {
			cmdStr := hook.Command
			if cmdStr == "" {
				cmdStr = fmt.Sprintf("script: %s", hook.Script)
			}
			ui.OutputLine("  %d. %s", i+1, hook.Name)
			ui.OutputLine("     Command: %s", cmdStr)
			ui.OutputLine("     Timeout: %s", hook.Timeout)
			ui.OutputLine("     On error: %s", hook.OnError)
			if len(hook.Env) > 0 {
				ui.OutputLine("     Environment:")
				for k, v := range hook.Env {
					ui.OutputLine("       %s=%s", k, v)
				}
			}
			ui.OutputLine("")
		}
	}

	if !hasHooks {
		ui.OutputLine("No hooks configured.")
		ui.OutputLine("Run 'amux hooks init' to create an example configuration.")
	} else if !trusted {
		ui.Warning("Hooks are configured but not trusted")
		ui.OutputLine("")
		ui.OutputLine("Run 'amux hooks trust' to enable them.")
	}

	return nil
}

func testHooks(cmd *cobra.Command, args []string) error {
	eventName := args[0]

	// Validate event
	event := hooks.Event(eventName)
	switch event {
	case hooks.EventWorkspaceCreate, hooks.EventWorkspaceRemove,
		hooks.EventSessionStart, hooks.EventSessionStop:
		// Valid event
	default:
		return fmt.Errorf("unknown event: %s", eventName)
	}

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("not in an amux project: %w", err)
	}

	configManager := config.NewManager(projectRoot)
	configDir := configManager.GetAmuxDir()

	// Load configuration
	hooksConfig, err := hooks.LoadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to load hooks configuration: %w", err)
	}

	// Get hooks for event
	eventHooks := hooksConfig.GetHooksForEvent(event)
	if len(eventHooks) == 0 {
		ui.OutputLine("No hooks configured for event '%s'", event)
		return nil
	}

	// Show what would be executed
	ui.OutputLine("Hooks that would run for '%s':", event)
	for i, hook := range eventHooks {
		cmdStr := hook.Command
		if cmdStr == "" {
			cmdStr = hook.Script
		}
		ui.OutputLine("  %d. \"%s\"", i+1, hook.Name)
		ui.OutputLine("     Would execute: %s", cmdStr)
		ui.OutputLine("     Timeout: %s", hook.Timeout)
		ui.OutputLine("     On error: %s", hook.OnError)
	}

	ui.OutputLine("")
	ui.OutputLine("Environment variables that would be set:")
	ui.OutputLine("  AMUX_EVENT=%s", event)
	ui.OutputLine("  AMUX_PROJECT_ROOT=%s", projectRoot)
	ui.OutputLine("  AMUX_CONFIG_DIR=%s", configDir)
	ui.OutputLine("  (Plus workspace/agent specific variables when applicable)")

	return nil
}
