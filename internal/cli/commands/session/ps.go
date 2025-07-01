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

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running sessions",
	Long: `List running sessions.

By default, shows only sessions in the current workspace.
Use --all to show sessions from all workspaces.`,
	RunE: ListSessions,
}

var psOpts struct {
	workspace string
	all       bool
	format    string
}

func init() {
	psCmd.Flags().StringVarP(&psOpts.workspace, "workspace", "w", "", "Filter by workspace")
	psCmd.Flags().BoolVarP(&psOpts.all, "all", "a", false, "Show sessions from all workspaces")
	psCmd.Flags().StringVarP(&psOpts.format, "format", "f", "", "Output format (json, wide)")
}

// BindPsFlags binds command flags to psOpts
func BindPsFlags(cmd *cobra.Command) {
	psOpts.workspace, _ = cmd.Flags().GetString("workspace")
	psOpts.all, _ = cmd.Flags().GetBool("all")
	psOpts.format, _ = cmd.Flags().GetString("format")
}

// SetPsAll sets the all flag
func SetPsAll(all bool) {
	psOpts.all = all
}

// SetPsFormat sets the format flag
func SetPsFormat(format string) {
	psOpts.format = format
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
	if !psOpts.all {
		if psOpts.workspace != "" {
			workspaceID = psOpts.workspace
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
	switch psOpts.format {
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
	headers := []string{"ID", "NAME", "STATUS", "RUNTIME", "WORKSPACE", "TASK", "CREATED"}

	var rows [][]string
	for _, s := range sessions {
		// Format status with exit code if available
		status := string(s.Status)
		if s.ExitCode != nil && *s.ExitCode != 0 {
			status = fmt.Sprintf("%s(%d)", status, *s.ExitCode)
		}

		// Runtime information
		runtime := s.Runtime
		if runtime == "" {
			runtime = "unknown"
		}

		// Task information (new)
		task := s.TaskName
		if task == "" {
			task = "-"
		}

		// Workspace
		workspace := s.WorkspaceID
		if workspace == "" {
			workspace = "-"
		}

		// Human-readable time
		created := formatDurationAgo(time.Since(s.StartedAt))

		// Session name (use ID for now as Name field doesn't exist)
		name := s.ID

		rows = append(rows, []string{
			s.ID,
			name,
			status,
			runtime,
			workspace,
			task,
			created,
		})
	}

	// Create and print table
	headerInterfaces := make([]interface{}, len(headers))
	for i, h := range headers {
		headerInterfaces[i] = h
	}
	tbl := ui.NewTable(headerInterfaces...)
	for _, row := range rows {
		tbl.AddRow(row[0], row[1], row[2], row[3], row[4], row[5], row[6])
	}
	tbl.Print()
}

// displaySessionsWide shows sessions with more details
func displaySessionsWide(sessions []*session.Session) {
	// Prepare table data
	headers := []string{"ID", "NAME", "STATUS", "RUNTIME", "WORKSPACE", "TASK", "PID", "STARTED", "DURATION", "COMMAND"}

	var rows [][]string
	for _, s := range sessions {
		// Format status with exit code if available
		status := string(s.Status)
		if s.ExitCode != nil && *s.ExitCode != 0 {
			status = fmt.Sprintf("%s(%d)", status, *s.ExitCode)
		}

		// Runtime information
		runtime := s.Runtime
		if runtime == "" {
			runtime = "unknown"
		}

		// Task information
		task := s.TaskName
		if task == "" {
			task = "-"
		}

		// Workspace
		workspace := s.WorkspaceID
		if workspace == "" {
			workspace = "-"
		}

		// Process ID
		pid := s.ProcessID
		if pid == "" {
			pid = "-"
		}

		// Started time
		started := s.StartedAt.Format("15:04:05")

		// Duration
		var duration string
		if s.StoppedAt != nil {
			duration = formatDuration(s.StoppedAt.Sub(s.StartedAt))
		} else {
			duration = formatDuration(time.Since(s.StartedAt))
		}

		// Command
		command := strings.Join(s.Command, " ")
		if command == "" {
			command = "-"
		}

		// Session name (use ID for now as Name field doesn't exist)
		name := s.ID

		rows = append(rows, []string{
			s.ID,
			name,
			status,
			runtime,
			workspace,
			task,
			pid,
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
		// Add all columns for wide format (10 columns total)
		tbl.AddRow(row[0], row[1], row[2], row[3], row[4], row[5], row[6], row[7], row[8], row[9])
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

// formatDurationAgo formats a duration with "ago" suffix for better readability
func formatDurationAgo(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}
