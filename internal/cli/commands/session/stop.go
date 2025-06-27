package session

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/hooks"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

func stopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <session>",
		Short: "Stop a running session",
		Args:  cobra.ExactArgs(1),
		RunE:  stopSession,
	}
}

func stopSession(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	// Get managers
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}
	wsManager, err := workspace.SetupManager(projectRoot)
	if err != nil {
		return err
	}
	sessionManager, err := session.SetupManagerWithWorkspace(projectRoot, wsManager)
	if err != nil {
		return err
	}

	// Get session
	sess, err := sessionManager.ResolveSession(cmd.Context(), session.Identifier(sessionID))
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Get workspace for hooks
	info := sess.Info()
	var ws *workspace.Workspace
	if info.WorkspaceID != "" {
		ws, _ = wsManager.ResolveWorkspace(cmd.Context(), workspace.Identifier(info.WorkspaceID))
	}

	// Execute session stop hooks (before stopping)
	if ws != nil {
		if err := executeSessionHooks(sess, ws, hooks.EventSessionStop); err != nil {
			ui.Warning("Hook execution failed: %v", err)
			// Continue with stop even if hooks fail
		}
	}

	// Stop session
	if err := sess.Stop(cmd.Context()); err != nil {
		return fmt.Errorf("failed to stop session: %w", err)
	}

	ui.OutputLine("Session %s stopped", sessionID)
	return nil
}
