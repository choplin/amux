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
	sessions, err := sessionManager.ListSessions(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		ui.Info("No active sessions found")
		return nil
	}

	// Update all session statuses in batch for better performance
	sessionManager.UpdateAllStatuses(cmd.Context(), sessions)

	// Create table
	tbl := ui.NewTable("SESSION", "NAME", "DESCRIPTION", "AGENT", "WORKSPACE", "STATUS", "IN STATUS", "TOTAL TIME")

	// Add rows
	for _, sess := range sessions {

		info := sess.Info()

		// Get workspace name
		ws, err := wsManager.ResolveWorkspace(cmd.Context(), workspace.Identifier(info.WorkspaceID))
		wsName := info.WorkspaceID
		if err == nil {
			wsName = ws.Name
		}

		// Calculate total time
		totalTime := "-"
		if info.StartedAt != nil {
			if info.StoppedAt != nil {
				totalTime = ui.FormatDuration(info.StoppedAt.Sub(*info.StartedAt))
			} else if info.StatusState.Status.IsRunning() {
				totalTime = ui.FormatDuration(time.Since(*info.StartedAt))
			}
		}

		// Format status for display
		statusStr := string(info.StatusState.Status)
		switch info.StatusState.Status {
		case session.StatusCreated:
			// StatusCreated uses default styling (no color)
		case session.StatusWorking:
			statusStr = ui.SuccessStyle.Render(statusStr)
		case session.StatusIdle:
			statusStr = ui.DimStyle.Render(statusStr)
		case session.StatusCompleted:
			statusStr = ui.InfoStyle.Render(statusStr)
		case session.StatusStopped:
			statusStr = ui.DimStyle.Render(statusStr)
		case session.StatusFailed:
			statusStr = ui.ErrorStyle.Render(statusStr)
		}

		// Show time in current status
		inStatusStr := "-"
		if !info.StatusState.StatusChangedAt.IsZero() {
			statusDuration := time.Since(info.StatusState.StatusChangedAt)
			inStatusStr = ui.FormatDuration(statusDuration)
		}

		displayID := info.ID
		if info.Index != "" {
			displayID = info.Index
		}

		// Format session name
		sessionName := info.Name

		// Format description with truncation
		description := info.Description
		if len(description) > 30 {
			description = description[:27] + "..."
		}

		tbl.AddRow(displayID, sessionName, description, info.AgentID, wsName, statusStr, inStatusStr, totalTime)
	}

	// Print with header
	ui.PrintSectionHeader("🤖", "Active Sessions", len(sessions))
	tbl.Print()
	ui.OutputLine("")

	return nil
}
