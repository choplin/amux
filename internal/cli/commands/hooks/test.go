package hooks

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/hooks"
)

var hooksTestCmd = &cobra.Command{
	Use:   "test <event>",
	Short: "Test hooks for a specific event",
	Long: `Test hooks for a specific event without actually running them.

This shows what hooks would be executed and with what environment
variables, but doesn't actually run the commands.`,
	Args: cobra.ExactArgs(1),
	RunE: testHooks,
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
