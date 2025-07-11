package session

import (
	"fmt"
	"os"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/runtime"
	"github.com/aki/amux/internal/session"
	"github.com/aki/amux/internal/workspace"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [-- command args...]",
	Short: "Run a task or command in a session",
	Long: `Run a task or command in a session.

You can either run a predefined task using --task flag, or specify a custom command after --.

Examples:
  # Run a predefined task
  amux session run --task dev
  amux session run -t dev

  # Run a custom command
  amux session run -- npm start

  # Run a task with arguments
  amux session run --task build -- --watch

  # Run in a specific workspace
  amux session run --task dev --workspace myworkspace

  # Run with tmux runtime
  amux session run --task dev --runtime tmux`,
	RunE: RunSession,
}

var runOpts struct {
	task        string
	workspace   string
	runtime     string
	environment []string
	workingDir  string
	follow      bool
	name        string
	description string
	enableLog   bool
}

func init() {
	runCmd.Flags().StringVarP(&runOpts.task, "task", "t", "", "Task name to run")
	runCmd.Flags().StringVarP(&runOpts.workspace, "workspace", "w", "", "Workspace to run in")
	runCmd.Flags().StringVarP(&runOpts.runtime, "runtime", "r", "local", "Runtime to use (local, local-detached, tmux)")
	runCmd.Flags().StringArrayVarP(&runOpts.environment, "env", "e", nil, "Environment variables (KEY=VALUE)")
	runCmd.Flags().StringVarP(&runOpts.workingDir, "dir", "d", "", "Working directory")
	runCmd.Flags().BoolVarP(&runOpts.follow, "follow", "f", false, "Follow logs")
	runCmd.Flags().StringVarP(&runOpts.name, "name", "n", "", "Human-readable name for the session")
	runCmd.Flags().StringVar(&runOpts.description, "description", "", "Description of session purpose")
	runCmd.Flags().BoolVar(&runOpts.enableLog, "log", false, "Enable logging to file (default: false)")
}

// BindRunFlags binds command flags to runOpts
func BindRunFlags(cmd *cobra.Command) {
	runOpts.task, _ = cmd.Flags().GetString("task")
	runOpts.workspace, _ = cmd.Flags().GetString("workspace")
	runOpts.runtime, _ = cmd.Flags().GetString("runtime")
	runOpts.environment, _ = cmd.Flags().GetStringArray("env")
	runOpts.workingDir, _ = cmd.Flags().GetString("dir")
	runOpts.follow, _ = cmd.Flags().GetBool("follow")
	runOpts.name, _ = cmd.Flags().GetString("name")
	runOpts.description, _ = cmd.Flags().GetString("description")
	runOpts.enableLog, _ = cmd.Flags().GetBool("log")
}

// RunSession implements the session run command
func RunSession(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Parse arguments
	taskName := runOpts.task
	var command []string

	// Validate that either task or command is specified, but not both
	if taskName != "" && len(args) > 0 {
		// If task is specified, args are passed to the task (after --)
		command = args
	} else if taskName == "" && len(args) > 0 {
		// Direct command execution
		command = args
	} else if taskName == "" && len(args) == 0 {
		return fmt.Errorf("either --task or command must be specified")
	}

	// Setup managers with project root detection
	configMgr, sessionMgr, err := setupManagers()
	if err != nil {
		return err
	}

	// Get workspace ID
	workspaceID := runOpts.workspace
	autoCreateWorkspace := false
	if workspaceID == "" {
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
		// If still no workspace, enable auto-creation
		if workspaceID == "" {
			autoCreateWorkspace = true
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

	// Create runtime options based on runtime type
	var runtimeOptions runtime.RuntimeOptions
	// Currently, no runtime-specific options are needed

	// For local runtime, show minimal messages
	showDetailedInfo := runOpts.runtime != "local"

	// Create session
	sess, err := sessionMgr.Create(ctx, session.CreateOptions{
		WorkspaceID:         workspaceID,
		AutoCreateWorkspace: autoCreateWorkspace,
		Name:                runOpts.name,
		Description:         runOpts.description,
		TaskName:            taskName,
		Command:             command,
		Runtime:             runOpts.runtime,
		Environment:         env,
		WorkingDir:          runOpts.workingDir,
		RuntimeOptions:      runtimeOptions,
		EnableLog:           runOpts.enableLog,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Show workspace creation message if applicable
	if autoCreateWorkspace && sess.WorkspaceID != "" {
		// Get workspace manager to resolve the workspace name
		wsMgr, err := workspace.SetupManager(configMgr.GetProjectRoot())
		if err == nil {
			ws, err := wsMgr.Get(ctx, workspace.ID(sess.WorkspaceID))
			if err == nil && ws.AutoCreated {
				ui.Success("Workspace created: %s", ws.Name)
			}
		}
	}

	// Display session information based on runtime type
	ui.Success("Session started: %s", sess.ID)

	if showDetailedInfo {
		// For detached runtimes, show full session information
		if sess.Name != "" {
			ui.Info("Name: %s", sess.Name)
		}
		if sess.Description != "" {
			ui.Info("Description: %s", sess.Description)
		}
		ui.Info("Runtime: %s", sess.Runtime)
		if sess.TaskName != "" {
			ui.Info("Task: %s", sess.TaskName)
		}
		if sess.WorkspaceID != "" {
			ui.Info("Workspace: %s", sess.WorkspaceID)
		}
	}

	// Provide appropriate feedback based on runtime
	if runOpts.runtime == "local-detached" || runOpts.runtime == "tmux" {
		ui.OutputLine("")
		ui.OutputLine("Running in detached mode")
		ui.OutputLine("Use 'amux session ps' to view status")
		ui.OutputLine("Use 'amux session attach %s' to attach", sess.ID)
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
