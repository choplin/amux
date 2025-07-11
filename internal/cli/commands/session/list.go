package session

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/session"
	"github.com/aki/amux/internal/workspace"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List running sessions",
	Aliases: []string{"ls", "ps"},
	Long: `List running sessions.

By default, shows only sessions in the current workspace.
Use --all to show sessions from all workspaces.`,
	RunE: ListSessions,
}

var listOpts struct {
	workspace string
	all       bool
	format    string
}

func init() {
	listCmd.Flags().StringVarP(&listOpts.workspace, "workspace", "w", "", "Filter by workspace")
	listCmd.Flags().BoolVarP(&listOpts.all, "all", "a", false, "Show sessions from all workspaces")
	listCmd.Flags().StringVarP(&listOpts.format, "format", "f", "", "Output format (json, wide)")
}

// BindListFlags binds command flags to listOpts
func BindListFlags(cmd *cobra.Command) {
	listOpts.workspace, _ = cmd.Flags().GetString("workspace")
	listOpts.all, _ = cmd.Flags().GetBool("all")
	listOpts.format, _ = cmd.Flags().GetString("format")
}

// SetListAll sets the all flag
func SetListAll(all bool) {
	listOpts.all = all
}

// SetListFormat sets the format flag
func SetListFormat(format string) {
	listOpts.format = format
}

// ListSessions implements the session ps command
func ListSessions(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Setup managers with project root detection
	configMgr, sessionMgr, err := setupManagers()
	if err != nil {
		return err
	}

	// Determine workspace filter
	workspaceID := ""
	if !listOpts.all {
		if listOpts.workspace != "" {
			workspaceID = listOpts.workspace
		} else {
			// Try to get current workspace
			wsMgr, err := workspace.SetupManager(configMgr.GetProjectRoot())
			if err == nil {
				// Check if we're in a workspace directory
				currentPath, _ := os.Getwd()
				workspaces, _ := wsMgr.List(ctx, workspace.ListOptions{})
				for _, ws := range workspaces {
					if currentPath == ws.Path {
						workspaceID = ws.ID
						break
					}
				}
			}
		}
	}

	// List sessions
	sessions, err := sessionMgr.List(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		if workspaceID != "" {
			ui.Info("No active sessions in workspace %s", workspaceID)
		} else {
			ui.Info("No active sessions")
		}
		return nil
	}

	// Display sessions
	switch listOpts.format {
	case "json":
		// JSON output
		for _, s := range sessions {
			data, _ := json.Marshal(s)
			fmt.Println(string(data))
		}
	case "wide":
		displaySessionsWide(sessions)
	default:
		displaySessions(sessions)
	}

	return nil
}

// displaySessions shows sessions in a table format
func displaySessions(sessions []*session.Session) {
	// Prepare table data
	headers := []string{"ID", "NAME", "STATUS", "LAST OUTPUT", "RUNTIME", "WORKSPACE", "DURATION"}

	var rows [][]string
	for _, s := range sessions {
		// Format status with exit code if available
		status := formatStatus(s.Status, s.ExitCode)

		// Format last output time
		lastOutput := formatLastOutput(s.LastActivityAt, s.Status)

		// Runtime information
		runtime := s.Runtime
		if runtime == "" {
			runtime = "local"
		}

		// Workspace - show just the name part if it's a full ID
		workspace := formatWorkspaceName(s.WorkspaceID)

		// Calculate duration
		var duration string
		if s.StoppedAt != nil {
			duration = formatDuration(s.StoppedAt.Sub(s.StartedAt))
		} else {
			duration = formatDuration(time.Since(s.StartedAt))
		}

		// Session name - use Name field if available, otherwise use ID
		name := s.Name
		if name == "" {
			name = s.ID
		}

		// Short ID - display short ID if available, otherwise first 8 chars of ID
		shortID := s.ShortID
		if shortID == "" {
			if len(s.ID) > 8 {
				shortID = s.ID[:8]
			} else {
				shortID = s.ID
			}
		}

		rows = append(rows, []string{
			shortID,
			name,
			status,
			lastOutput,
			runtime,
			workspace,
			duration,
		})
	}

	// Create and print table
	headerInterfaces := make([]interface{}, len(headers))
	for i, h := range headers {
		headerInterfaces[i] = h
	}
	tbl := ui.NewTable(headerInterfaces...)
	for _, row := range rows {
		// Convert row slice to interface slice for AddRow
		rowInterfaces := make([]interface{}, len(row))
		for i, v := range row {
			rowInterfaces[i] = v
		}
		tbl.AddRow(rowInterfaces...)
	}
	tbl.Print()
}

// displaySessionsWide shows sessions with more details
func displaySessionsWide(sessions []*session.Session) {
	// Prepare table data
	headers := []string{"SESSION", "NAME", "DESCRIPTION", "STATUS", "LAST OUTPUT", "RUNTIME", "WORKSPACE", "TASK", "STARTED", "DURATION", "COMMAND"}

	var rows [][]string
	for _, s := range sessions {
		// Format status with color coding
		status := formatStatus(s.Status, s.ExitCode)

		// Format last output time
		lastOutput := formatLastOutput(s.LastActivityAt, s.Status)

		// Runtime information
		runtime := s.Runtime
		if runtime == "" {
			runtime = "local"
		}

		// Task information
		task := s.TaskName
		if task == "" {
			task = "-"
		}

		// Workspace - show readable name
		workspace := formatWorkspaceName(s.WorkspaceID)

		// Started time
		started := s.StartedAt.Format("15:04:05")

		// Duration
		var duration string
		if s.StoppedAt != nil {
			duration = formatDuration(s.StoppedAt.Sub(s.StartedAt))
		} else {
			duration = formatDuration(time.Since(s.StartedAt))
		}

		// Command - truncate if too long
		command := strings.Join(s.Command, " ")
		if command == "" {
			command = "-"
		} else if len(command) > 50 {
			command = command[:47] + "..."
		}

		// Session name - use Name field if available, otherwise use ID
		name := s.Name
		if name == "" {
			name = s.ID
		}

		// Description - truncate if too long
		description := s.Description
		if description == "" {
			description = "-"
		} else if len(description) > 40 {
			description = description[:37] + "..."
		}

		rows = append(rows, []string{
			s.ID,
			name,
			description,
			status,
			lastOutput,
			runtime,
			workspace,
			task,
			started,
			duration,
			command,
		})
	}

	// Create and print table
	headerInterfaces := make([]interface{}, len(headers))
	for i, h := range headers {
		headerInterfaces[i] = h
	}
	tbl := ui.NewTable(headerInterfaces...)
	for _, row := range rows {
		// Convert row slice to interface slice for AddRow
		rowInterfaces := make([]interface{}, len(row))
		for i, v := range row {
			rowInterfaces[i] = v
		}
		tbl.AddRow(rowInterfaces...)
	}
	tbl.Print()
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// formatStatus formats the session status with color coding
func formatStatus(status session.Status, exitCode *int) string {
	statusStr := string(status)

	// Add exit code if non-zero
	if exitCode != nil && *exitCode != 0 {
		statusStr = fmt.Sprintf("%s(%d)", statusStr, *exitCode)
	}

	// Apply color based on status
	switch status {
	case session.StatusStarting:
		return ui.InfoStyle.Render(statusStr)
	case session.StatusRunning:
		return ui.SuccessStyle.Render(statusStr)
	case session.StatusStopped:
		return ui.DimStyle.Render(statusStr)
	case session.StatusFailed:
		return ui.ErrorStyle.Render(statusStr)
	case session.StatusUnknown:
		return ui.DimStyle.Render(statusStr)
	default:
		return statusStr
	}
}

// formatWorkspaceName extracts a readable name from workspace ID
func formatWorkspaceName(workspaceID string) string {
	if workspaceID == "" {
		return "-"
	}

	// If it's a full workspace ID like "workspace-fix-session-project--1751273433-e6f91ac7"
	// Extract the meaningful part
	if strings.HasPrefix(workspaceID, "workspace-") {
		parts := strings.SplitN(workspaceID[10:], "--", 2)
		if len(parts) > 0 {
			return parts[0]
		}
	}

	return workspaceID
}

// formatLastOutput formats the time elapsed since last output
func formatLastOutput(lastActivityAt time.Time, status session.Status) string {
	// If session is not running, return "-"
	if status != session.StatusRunning {
		return ui.DimStyle.Render("-")
	}

	// If lastActivityAt is zero (never had output), return "never"
	if lastActivityAt.IsZero() {
		return ui.DimStyle.Render("never")
	}

	// Calculate elapsed time
	elapsed := time.Since(lastActivityAt)

	// Format the duration in a human-readable way
	var formatted string
	switch {
	case elapsed < time.Second:
		formatted = "now"
	case elapsed < time.Minute:
		formatted = fmt.Sprintf("%ds", int(elapsed.Seconds()))
	case elapsed < time.Hour:
		formatted = fmt.Sprintf("%dm", int(elapsed.Minutes()))
	case elapsed < 24*time.Hour:
		formatted = fmt.Sprintf("%dh", int(elapsed.Hours()))
	default:
		formatted = fmt.Sprintf("%dd", int(elapsed.Hours()/24))
	}

	// Return without any color styling
	return formatted
}
