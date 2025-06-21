package hooks

import (
	"fmt"
	"os/user"
	"time"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/hooks"
)

var hooksTrustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Trust hooks in this project",
	Long: `Trust the hooks configured in this project.

For security, hooks must be explicitly trusted before they will run.
This command shows you all configured hooks and asks for confirmation.`,
	RunE: trustHooks,
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

	ui.OutputLine("Hooks trusted. They will now run automatically during workspace operations.")
	return nil
}
