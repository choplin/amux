package session

import (
	"fmt"
	"os"
	"time"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/config"
	"github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/runtime/local"
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
	detach      bool
}

func init() {
	runCmd.Flags().StringVarP(&runOpts.workspace, "workspace", "w", "", "Workspace to run in")
	runCmd.Flags().StringVarP(&runOpts.runtime, "runtime", "r", "local", "Runtime to use (local, tmux)")
	runCmd.Flags().StringArrayVarP(&runOpts.environment, "env", "e", nil, "Environment variables (KEY=VALUE)")
	runCmd.Flags().StringVarP(&runOpts.workingDir, "dir", "d", "", "Working directory")
	runCmd.Flags().BoolVarP(&runOpts.follow, "follow", "f", false, "Follow logs")
	runCmd.Flags().BoolVar(&runOpts.detach, "detach", false, "Run in background (detached mode, local runtime only)")
}

// BindRunFlags binds command flags to runOpts
func BindRunFlags(cmd *cobra.Command) {
	runOpts.workspace, _ = cmd.Flags().GetString("workspace")
	runOpts.runtime, _ = cmd.Flags().GetString("runtime")
	runOpts.environment, _ = cmd.Flags().GetStringArray("env")
	runOpts.workingDir, _ = cmd.Flags().GetString("dir")
	runOpts.follow, _ = cmd.Flags().GetBool("follow")
	runOpts.detach, _ = cmd.Flags().GetBool("detach")
}

// RunSession implements the session run command
func RunSession(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Parse arguments
	var taskName string
	var command []string

	// Cobra processes -- specially and removes it from args
	// If we have args and they don't look like a task name, assume they are a command
	if len(args) > 0 {
		// Check if this might be a direct command (not a task)
		// In Cobra, after --, all args are passed through
		if cmd.ArgsLenAtDash() != -1 {
			// -- was present
			dashPos := cmd.ArgsLenAtDash()
			if dashPos > 0 {
				// Task name before --
				taskName = args[0]
				command = args[dashPos:]
			} else {
				// No task name, just command after --
				command = args
			}
		} else {
			// No -- present, first arg is task name
			taskName = args[0]
		}
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

	// Validate detach flag is only used with local runtime
	if runOpts.detach && runOpts.runtime != "local" {
		return fmt.Errorf("--detach flag is only supported for local runtime")
	}

	// Create runtime options based on runtime type
	var runtimeOptions runtime.RuntimeOptions
	if runOpts.runtime == "local" {
		runtimeOptions = local.Options{
			Detach: runOpts.detach,
		}
	}

	// Create session
	sess, err := sessionMgr.Create(ctx, session.CreateOptions{
		WorkspaceID:    workspaceID,
		TaskName:       taskName,
		Command:        command,
		Runtime:        runOpts.runtime,
		Environment:    env,
		WorkingDir:     runOpts.workingDir,
		RuntimeOptions: runtimeOptions,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	ui.Success("Session started: %s", sess.ID)
	ui.OutputLine("Runtime: %s", sess.Runtime)
	if sess.WorkspaceID != "" {
		ui.OutputLine("Workspace: %s", sess.WorkspaceID)
	}

	// Provide appropriate feedback based on mode
	if runOpts.detach {
		ui.OutputLine("")
		ui.OutputLine("Running in detached mode")
		ui.OutputLine("Use 'amux session ps' to view status")
		ui.OutputLine("Use 'amux session attach %s' to attach", sess.ID)
	} else {
		// For foreground mode, wait for the session to complete
		// Get the session to access the process
		updatedSess, err := sessionMgr.Get(ctx, sess.ID)
		if err != nil {
			return fmt.Errorf("failed to get session: %w", err)
		}

		// Wait for the process to complete
		if updatedSess.Status == session.StatusRunning {
			// Note: We need a way to wait for the process
			// For now, we'll use a simple polling approach
			for {
				time.Sleep(100 * time.Millisecond)
				s, err := sessionMgr.Get(ctx, sess.ID)
				if err != nil {
					return fmt.Errorf("failed to get session status: %w", err)
				}
				if s.Status != session.StatusRunning {
					// Process completed
					if s.ExitCode != nil && *s.ExitCode != 0 {
						return fmt.Errorf("process exited with code %d", *s.ExitCode)
					}
					break
				}
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
