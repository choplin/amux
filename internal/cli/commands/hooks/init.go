package hooks

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/hooks"
)

var hooksInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize hooks configuration",
	Long: `Initialize a hooks configuration file with example hooks.

This creates a .amux/hooks.yaml file with commented examples showing
how to configure hooks for different events.`,
	RunE: initHooks,
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

	ui.OutputLine("Created hooks configuration at %s", hooksPath)
	ui.OutputLine("Edit this file to configure your hooks, then run 'amux hooks trust' to enable them.")

	return nil
}
