package hooks

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/hooks"
)

var hooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured hooks",
	Long:  `List all hooks configured in this project.`,
	RunE:  listHooks,
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
