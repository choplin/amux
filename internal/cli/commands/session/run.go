package session

import (
	"fmt"
	"os"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/config"
	"github.com/aki/amux/internal/session"
	"github.com/aki/amux/internal/workspace"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [task-name] [-- command args...]",
	Short: "Run a task or command in a session",
	Long: `Run a task or command in a session.

You can either run a predefined task by name, or specify a custom command after --.

Examples:
  # Run a predefined task
  amux session run dev

  # Run a custom command
  amux session run -- npm start

  # Run in a specific workspace
  amux session run dev --workspace myworkspace

  # Run with tmux runtime
  amux session run dev --runtime tmux`,
	RunE: RunSession,
}

var runOpts struct {
	workspace   string
	runtime     string
	environment []string
	workingDir  string
	follow      bool
}

func init() {
	runCmd.Flags().StringVarP(&runOpts.workspace, "workspace", "w", "", "Workspace to run in")
	runCmd.Flags().StringVarP(&runOpts.runtime, "runtime", "r", "local", "Runtime to use (local, tmux)")
	runCmd.Flags().StringArrayVarP(&runOpts.environment, "env", "e", nil, "Environment variables (KEY=VALUE)")
	runCmd.Flags().StringVarP(&runOpts.workingDir, "dir", "d", "", "Working directory")
	runCmd.Flags().BoolVarP(&runOpts.follow, "follow", "f", false, "Follow logs")
}

// BindRunFlags binds command flags to runOpts
func BindRunFlags(cmd *cobra.Command) {
	runOpts.workspace, _ = cmd.Flags().GetString("workspace")
	runOpts.runtime, _ = cmd.Flags().GetString("runtime")
	runOpts.environment, _ = cmd.Flags().GetStringArray("env")
	runOpts.workingDir, _ = cmd.Flags().GetString("dir")
	runOpts.follow, _ = cmd.Flags().GetBool("follow")
}

// RunSession implements the session run command
func RunSession(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Parse arguments
	var taskName string
	var command []string

	dashIndex := -1
	for i, arg := range args {
		if arg == "--" {
			dashIndex = i
			break
		}
	}

	if dashIndex >= 0 {
		// Command specified after --
		if dashIndex > 0 {
			taskName = args[0]
		}
		command = args[dashIndex+1:]
	} else if len(args) > 0 {
		// Task name specified
		taskName = args[0]
	} else {
		return fmt.Errorf("either task name or command must be specified")
	}

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

	// Get workspace ID
	workspaceID := runOpts.workspace
	if workspaceID == "" {
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

	// Parse environment variables
	env := make(map[string]string)
	for _, e := range runOpts.environment {
		parts := splitKeyValue(e)
		if len(parts) != 2 {
			return fmt.Errorf("invalid environment variable: %s", e)
		}
		env[parts[0]] = parts[1]
	}

	// Get session manager
	sessionMgr := getSessionManager(configMgr)

	// Create session
	sess, err := sessionMgr.Create(ctx, session.CreateOptions{
		WorkspaceID: workspaceID,
		TaskName:    taskName,
		Command:     command,
		Runtime:     runOpts.runtime,
		Environment: env,
		WorkingDir:  runOpts.workingDir,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	ui.Success("Session started: %s", sess.ID)
	ui.Info("Runtime: %s", sess.Runtime)
	if sess.WorkspaceID != "" {
		ui.Info("Workspace: %s", sess.WorkspaceID)
	}

	// Follow logs if requested
	if runOpts.follow {
		ui.Info("Following logs...")
		reader, err := sessionMgr.Logs(ctx, sess.ID, true)
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}
		defer reader.Close()

		// Copy logs to stdout
		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}

	return nil
}

// splitKeyValue splits a KEY=VALUE string
func splitKeyValue(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
