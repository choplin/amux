package workspace

import (
	"fmt"
	"time"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/hooks"
	"github.com/aki/amux/internal/core/workspace"
)

// executeWorkspaceHooks runs hooks for the given workspace event
func executeWorkspaceHooks(ws *workspace.Workspace, event hooks.Event) error {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	configManager := config.NewManager(projectRoot)
	configDir := configManager.GetAmuxDir()

	// Load hooks configuration
	hooksConfig, err := hooks.LoadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to load hooks: %w", err)
	}

	// Get hooks for this event
	eventHooks := hooksConfig.GetHooksForEvent(event)
	if len(eventHooks) == 0 {
		return nil // No hooks configured
	}

	// Check if hooks are trusted
	trusted, err := hooks.IsTrusted(configDir, hooksConfig)
	if err != nil {
		return fmt.Errorf("failed to check hook trust: %w", err)
	}

	if !trusted {
		ui.Warning("This project has hooks configured but they are not trusted")
		ui.OutputLine("Run 'amux hooks trust' to trust hooks in this project.")
		return nil
	}

	// Prepare environment variables
	env := map[string]string{
		"AMUX_WORKSPACE_ID":          ws.ID,
		"AMUX_WORKSPACE_NAME":        ws.Name,
		"AMUX_WORKSPACE_PATH":        ws.Path,
		"AMUX_WORKSPACE_BRANCH":      ws.Branch,
		"AMUX_WORKSPACE_BASE_BRANCH": ws.BaseBranch,
		"AMUX_EVENT":                 string(event),
		"AMUX_EVENT_TIME":            time.Now().Format(time.RFC3339),
		"AMUX_PROJECT_ROOT":          projectRoot,
		"AMUX_CONFIG_DIR":            configDir,
	}

	// Execute hooks in workspace directory
	executor := hooks.NewExecutor(configDir, env).WithWorkingDir(ws.Path)
	return executor.ExecuteHooks(event, eventHooks)
}
