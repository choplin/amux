package session

import (
	"fmt"
	"time"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/hooks"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

// executeSessionHooks executes hooks for session events
func executeSessionHooks(sess session.Session, ws *workspace.Workspace, event hooks.Event) error {
	if ws == nil {
		return fmt.Errorf("session hooks require workspace assignment")
	}

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

	// Get session info
	info := sess.Info()

	// Prepare environment variables
	env := map[string]string{
		// Session-specific variables
		"AMUX_SESSION_ID":          info.ID,
		"AMUX_SESSION_INDEX":       info.Index,
		"AMUX_SESSION_AGENT_ID":    info.AgentID,
		"AMUX_SESSION_NAME":        info.Name,
		"AMUX_SESSION_DESCRIPTION": info.Description,
		"AMUX_SESSION_COMMAND":     info.Command,
		// Workspace variables
		"AMUX_WORKSPACE_ID":          ws.ID,
		"AMUX_WORKSPACE_NAME":        ws.Name,
		"AMUX_WORKSPACE_PATH":        ws.Path,
		"AMUX_WORKSPACE_BRANCH":      ws.Branch,
		"AMUX_WORKSPACE_BASE_BRANCH": ws.BaseBranch,
		// Event and context
		"AMUX_EVENT":        string(event),
		"AMUX_EVENT_TIME":   time.Now().Format(time.RFC3339),
		"AMUX_PROJECT_ROOT": projectRoot,
		"AMUX_CONFIG_DIR":   configDir,
	}

	// Execute hooks in workspace directory
	executor := hooks.NewExecutor(configDir, env).WithWorkingDir(ws.Path)
	return executor.ExecuteHooks(event, eventHooks)
}
