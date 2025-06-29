package session

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/config"
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

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create config manager
	configMgr := config.NewManager(wd)
	if !configMgr.IsInitialized() {
		return fmt.Errorf("amux not initialized. Run 'amux init' first")
	}

	// Determine workspace filter
	workspaceID := ""
	if !psOpts.all {
		if psOpts.workspace != "" {
			workspaceID = psOpts.workspace
		} else {
			// Try to get current workspace
			wsMgr, err := workspace.SetupManager(wd)
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

	// Get session manager
	sessionMgr := getSessionManager(configMgr)

	// List sessions
	sessions, err := sessionMgr.List(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		ui.Info("No sessions found")
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
	// Header
	fmt.Printf("%-8s %-12s %-10s %-8s %-10s %s\n",
		"ID", "WORKSPACE", "RUNTIME", "STATUS", "STARTED", "COMMAND")
	fmt.Println(strings.Repeat("-", 70))

	// Rows
	for _, s := range sessions {
		workspace := s.WorkspaceID
		if workspace == "" {
			workspace = "-"
		} else if len(workspace) > 10 {
			workspace = workspace[:10]
		}

		started := time.Since(s.StartedAt).Round(time.Second)
		command := strings.Join(s.Command, " ")
		if len(command) > 30 {
			command = command[:27] + "..."
		}

		fmt.Printf("%-8s %-12s %-10s %-8s %-10s %s\n",
			s.ID, workspace, s.Runtime, s.Status, formatDuration(started), command)
	}
}

// displaySessionsWide shows sessions with more details
func displaySessionsWide(sessions []*session.Session) {
	// Header
	fmt.Printf("%-8s %-12s %-10s %-8s %-8s %-10s %-10s %s\n",
		"ID", "WORKSPACE", "RUNTIME", "STATUS", "PID", "STARTED", "DURATION", "COMMAND")
	fmt.Println(strings.Repeat("-", 90))

	// Rows
	for _, s := range sessions {
		workspace := s.WorkspaceID
		if workspace == "" {
			workspace = "-"
		} else if len(workspace) > 10 {
			workspace = workspace[:10]
		}

		pid := s.ProcessID
		if pid == "" {
			pid = "-"
		}

		started := s.StartedAt.Format("15:04:05")

		var duration string
		if s.StoppedAt != nil {
			duration = s.StoppedAt.Sub(s.StartedAt).Round(time.Second).String()
		} else {
			duration = time.Since(s.StartedAt).Round(time.Second).String()
		}

		command := strings.Join(s.Command, " ")
		if len(command) > 40 {
			command = command[:37] + "..."
		}

		fmt.Printf("%-8s %-12s %-10s %-8s %-8s %-10s %-10s %s\n",
			s.ID, workspace, s.Runtime, s.Status, pid, started, duration, command)
	}
}

// formatDuration formats a duration for display
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	} else {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
