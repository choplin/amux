package session

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls", "ps"},
		Short:   "List active sessions",
		Long: `List all agent sessions.

Shows session ID, agent, workspace, status, and runtime.`,
		RunE: listSessions,
	}
}

func listSessions(cmd *cobra.Command, args []string) error {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	// Create managers
	configManager := config.NewManager(projectRoot)
	wsManager, err := workspace.NewManager(configManager)
	if err != nil {
		return fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Create session manager
	sessionManager, err := createSessionManager(configManager, wsManager)
	if err != nil {
		return err
	}

	// List sessions
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		ui.Info("No active sessions found")
		return nil
	}

	// Create table
	tbl := ui.NewTable("SESSION", "AGENT", "WORKSPACE", "STATUS", "RUNTIME")

	// Add rows
	for _, sess := range sessions {
		info := sess.Info()

		// Get workspace name
		ws, err := wsManager.ResolveWorkspace(info.WorkspaceID)
		wsName := info.WorkspaceID
		if err == nil {
			wsName = ws.Name
		}

		// Calculate runtime
		runtime := "-"
		if info.StartedAt != nil {
			if info.StoppedAt != nil {
				runtime = ui.FormatDuration(info.StoppedAt.Sub(*info.StartedAt))
			} else if info.Status == session.StatusRunning {
				runtime = ui.FormatDuration(time.Since(*info.StartedAt))
			}
		}

		// Format status for display
		statusStr := string(info.Status)
		switch info.Status {
		case session.StatusCreated:
			// StatusCreated uses default styling (no color)
		case session.StatusRunning:
			statusStr = ui.SuccessStyle.Render(statusStr)
		case session.StatusStopped:
			statusStr = ui.DimStyle.Render(statusStr)
		case session.StatusFailed:
			statusStr = ui.ErrorStyle.Render(statusStr)
		}

		displayID := info.ID
		if info.Index != "" {
			displayID = info.Index
		}

		tbl.AddRow(displayID, info.AgentID, wsName, statusStr, runtime)
	}

	// Print with header
	ui.PrintSectionHeader("🤖", "Active Sessions", len(sessions))
	tbl.Print()
	ui.OutputLine("")

	return nil
}
