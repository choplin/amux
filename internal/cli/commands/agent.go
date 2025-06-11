package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/agent"
	"github.com/aki/amux/internal/core/common"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

var (
	// Flags for agent run
	runWorkspace string
	runCommand   string
	runEnv       []string
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage AI agent sessions",
	Long: `Manage AI agent sessions in multiplexed workspaces.

Run multiple AI agents concurrently in isolated workspaces,
attach to running sessions, and manage agent lifecycle.`,
}

var agentRunCmd = &cobra.Command{
	Use:   "run <agent>",
	Short: "Run an AI agent in a workspace",
	Long: `Run an AI agent in a workspace.

Examples:
  # Run Claude in the latest workspace
  amux agent run claude

  # Run Claude in a specific workspace
  amux agent run claude --workspace feature-auth

  # Run with custom command
  amux agent run claude --command "claude code --model opus"

  # Run with environment variables
  amux agent run claude --env ANTHROPIC_API_KEY=sk-...`,
	Args: cobra.ExactArgs(1),
	RunE: runAgent,
}

var agentListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List agent sessions",
	Long: `List all agent sessions.

Shows session ID, agent, workspace, status, and runtime.`,
	RunE: listAgents,
}

var agentAttachCmd = &cobra.Command{
	Use:   "attach <session>",
	Short: "Attach to a running agent session",
	Long: `Attach to a running agent session.

This will connect you to the tmux session where the agent is running.
Use Ctrl-B D to detach from the session without stopping it.`,
	Args: cobra.ExactArgs(1),
	RunE: attachAgent,
}

var agentStopCmd = &cobra.Command{
	Use:   "stop <session>",
	Short: "Stop a running agent session",
	Args:  cobra.ExactArgs(1),
	RunE:  stopAgent,
}

var agentLogsCmd = &cobra.Command{
	Use:   "logs <session>",
	Short: "View agent session output",
	Long: `View the output from an agent session.

Shows the current content of the agent's terminal.`,
	Args: cobra.ExactArgs(1),
	RunE: viewAgentLogs,
}

func init() {
	// Add subcommands
	agentCmd.AddCommand(agentRunCmd)
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentAttachCmd)
	agentCmd.AddCommand(agentStopCmd)
	agentCmd.AddCommand(agentLogsCmd)

	// Run command flags
	agentRunCmd.Flags().StringVarP(&runWorkspace, "workspace", "w", "", "Workspace to run agent in (name or ID)")
	agentRunCmd.Flags().StringVarP(&runCommand, "command", "c", "", "Override agent command")
	agentRunCmd.Flags().StringSliceVarP(&runEnv, "env", "e", []string{}, "Environment variables (KEY=VALUE)")
}

func runAgent(cmd *cobra.Command, args []string) error {
	agentID := args[0]

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

	// Get or select workspace
	var ws *workspace.Workspace
	if runWorkspace != "" {
		ws, err = wsManager.ResolveWorkspace(runWorkspace)
		if err != nil {
			return fmt.Errorf("failed to resolve workspace: %w", err)
		}
	} else {
		// Get the most recent workspace
		workspaces, err := wsManager.List(workspace.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list workspaces: %w", err)
		}
		if len(workspaces) == 0 {
			return fmt.Errorf("no workspaces available. Create one with 'amux ws create'")
		}
		ws = workspaces[0] // List returns sorted by UpdatedAt desc
	}

	// Create agent manager
	agentManager := agent.NewManager(configManager)

	// Get agent configuration
	agentConfig, _ := agentManager.GetAgent(agentID)

	// Determine command to use
	command := runCommand
	if command == "" {
		// Use agent's configured command or fall back to agent ID
		command, _ = agentManager.GetDefaultCommand(agentID)
	}

	// Merge environment variables
	env := make(map[string]string)

	// First, add agent's default environment
	if agentConfig != nil && agentConfig.Environment != nil {
		for k, v := range agentConfig.Environment {
			env[k] = v
		}
	}

	// Then, override with command-line environment
	for _, envVar := range runEnv {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid environment variable format: %s (expected KEY=VALUE)", envVar)
		}
		env[parts[0]] = parts[1]
	}

	// Get ID mapper (workspace manager already has it internally)
	idMapper, err := common.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return fmt.Errorf("failed to create ID mapper: %w", err)
	}

	// Create session manager
	sessionManager, err := createSessionManager(configManager, wsManager, idMapper)
	if err != nil {
		return err
	}

	// Create session
	opts := session.SessionOptions{
		WorkspaceID: ws.ID,
		AgentID:     agentID,
		Command:     command,
		Environment: env,
	}

	sess, err := sessionManager.CreateSession(opts)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	sessionID := sess.ID()
	if info := sess.Info(); info.Index != "" {
		sessionID = info.Index
	}
	ui.Info("Created session %s for agent '%s' in workspace '%s'", sessionID, agentID, ws.Name)

	// Start session
	ctx := context.Background()
	if err := sess.Start(ctx); err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	ui.Success("Agent session started successfully!")

	// Show attach instruction if tmux session
	info := sess.Info()
	if info.TmuxSession != "" {
		ui.Info("To attach to this session, run:")
		ui.Info("  tmux attach-session -t %s", info.TmuxSession)
		attachID := sess.ID()
		if info.Index != "" {
			attachID = info.Index
		}
		ui.Info("  or: amux attach %s", attachID)
	}

	return nil
}

func listAgents(cmd *cobra.Command, args []string) error {
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

	// Get ID mapper (workspace manager already has it internally)
	idMapper, err := common.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return fmt.Errorf("failed to create ID mapper: %w", err)
	}

	// Create session manager
	sessionManager, err := createSessionManager(configManager, wsManager, idMapper)
	if err != nil {
		return err
	}

	// List sessions
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		ui.Info("No agent sessions found")
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
	ui.PrintSectionHeader("🤖", "Agent Sessions", len(sessions))
	tbl.Print()
	fmt.Println()

	return nil
}

func attachAgent(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

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

	// Get ID mapper (workspace manager already has it internally)
	idMapper, err := common.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return fmt.Errorf("failed to create ID mapper: %w", err)
	}

	// Create session manager
	sessionManager, err := createSessionManager(configManager, wsManager, idMapper)
	if err != nil {
		return err
	}

	// Get session
	sess, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Check if running
	if sess.Status() != session.StatusRunning {
		return fmt.Errorf("session is not running (status: %s)", sess.Status())
	}

	// Get tmux session name
	info := sess.Info()
	if info.TmuxSession == "" {
		return fmt.Errorf("session does not have a tmux session")
	}

	// Execute tmux attach
	ui.Info("Attaching to session %s...", sessionID)
	tmuxCmd := exec.Command("tmux", "attach-session", "-t", info.TmuxSession)
	tmuxCmd.Stdin = os.Stdin
	tmuxCmd.Stdout = os.Stdout
	tmuxCmd.Stderr = os.Stderr

	return tmuxCmd.Run()
}

func stopAgent(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

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

	// Get ID mapper (workspace manager already has it internally)
	idMapper, err := common.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return fmt.Errorf("failed to create ID mapper: %w", err)
	}

	// Create session manager
	sessionManager, err := createSessionManager(configManager, wsManager, idMapper)
	if err != nil {
		return err
	}

	// Get session
	sess, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Stop session
	if err := sess.Stop(); err != nil {
		return fmt.Errorf("failed to stop session: %w", err)
	}

	ui.Success("Session %s stopped", sessionID)
	return nil
}

func viewAgentLogs(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

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

	// Get ID mapper (workspace manager already has it internally)
	idMapper, err := common.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return fmt.Errorf("failed to create ID mapper: %w", err)
	}

	// Create session manager
	sessionManager, err := createSessionManager(configManager, wsManager, idMapper)
	if err != nil {
		return err
	}

	// Get session
	sess, err := sessionManager.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Get output
	output, err := sess.GetOutput()
	if err != nil {
		return fmt.Errorf("failed to get session output: %w", err)
	}

	// Print output
	fmt.Print(string(output))
	return nil
}
