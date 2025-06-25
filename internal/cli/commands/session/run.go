package session

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/agent"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/hooks"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

func runCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <agent>",
		Short: "Run an agent session in a workspace",
		Long: `Run an agent session in a workspace.

If no workspace is specified, a new workspace will be automatically created
with a name based on the session ID (e.g., session-f47ac10b).

Examples:
  # Run Claude with auto-created workspace
  amux session run claude
  # Run Claude with custom workspace name and description
  amux session run claude --name feature-auth --description "Implementing authentication"
  # Run Claude in a specific workspace
  amux session run claude --workspace feature-auth
  # Run with custom command
  amux session run claude --command "claude code --model opus"
  # Run with environment variables
  amux session run claude --env ANTHROPIC_API_KEY=sk-...
  # Run with initial prompt
  amux session run claude --initial-prompt "Please analyze the codebase"`,
		Args: cobra.ExactArgs(1),
		RunE: runSession,
	}

	// Run command flags
	cmd.Flags().StringVarP(&runWorkspace, "workspace", "w", "", "Workspace to run agent in (name or ID)")
	cmd.Flags().StringVarP(&runCommand, "command", "c", "", "Override agent command")
	cmd.Flags().StringSliceVarP(&runEnv, "env", "e", []string{}, "Environment variables (KEY=VALUE)")
	cmd.Flags().StringVarP(&runInitialPrompt, "initial-prompt", "p", "", "Initial prompt to send to the agent after starting")
	cmd.Flags().StringVarP(&runSessionName, "session-name", "", "", "Human-readable name for the session")
	cmd.Flags().StringVarP(&runSessionDescription, "session-description", "", "", "Description of the session purpose")
	cmd.Flags().StringVarP(&runName, "name", "n", "", "Name for the auto-created workspace (only used when --workspace is not specified)")
	cmd.Flags().StringVarP(&runDescription, "description", "d", "", "Description for the auto-created workspace (only used when --workspace is not specified)")

	return cmd
}

func runSession(cmd *cobra.Command, args []string) error {
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

	// Generate session ID upfront
	sessionID := session.GenerateID()

	// Get or select workspace
	var ws *workspace.Workspace
	if runWorkspace != "" {
		ws, err = wsManager.ResolveWorkspace(cmd.Context(), workspace.Identifier(runWorkspace))
		if err != nil {
			return fmt.Errorf("failed to resolve workspace: %w", err)
		}
	} else {
		// Auto-create a new workspace using session ID
		ws, err = createAutoWorkspace(cmd.Context(), wsManager, sessionID, runName, runDescription)
		if err != nil {
			return fmt.Errorf("failed to create auto-workspace: %w", err)
		}
		ui.Success("Workspace created successfully: %s", ws.Name)
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

	// Create session manager
	sessionManager, err := createSessionManager(configManager, wsManager)
	if err != nil {
		return err
	}

	// Determine session type from agent config
	sessionType := session.TypeTmux // Default
	if agentConfig != nil {
		// Convert agent type to session type
		switch agentConfig.Type {
		case config.AgentTypeTmux:
			sessionType = session.TypeTmux
		case config.AgentTypeClaudeCode, config.AgentTypeAPI:
			// Future: add more type mappings as needed
		}
	}

	// Create session
	opts := session.Options{
		ID:            sessionID,
		Type:          sessionType,
		WorkspaceID:   ws.ID,
		AgentID:       agentID,
		Command:       command,
		Environment:   env,
		InitialPrompt: runInitialPrompt,
		Name:          runSessionName,
		Description:   runSessionDescription,
	}

	sess, err := sessionManager.CreateSession(cmd.Context(), opts)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	displayID := sess.ID()
	if info := sess.Info(); info.Index != "" {
		displayID = info.Index
	}

	// Start session
	if err := sess.Start(cmd.Context()); err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	ui.Success("Session started successfully")
	ui.OutputLine("")
	ui.PrintKeyValue("Session", displayID)
	ui.PrintKeyValue("Workspace", ws.Name)
	ui.PrintKeyValue("Agent", agentID)

	// Execute session start hooks
	if err := executeSessionHooks(sess, ws, hooks.EventSessionStart); err != nil {
		ui.Error("Hook execution failed: %v", err)
		// Don't fail the session start, just warn
	}

	// Handle auto-attach for tmux sessions
	info := sess.Info()
	if info.TmuxSession != "" {
		// Check if we should auto-attach based on agent config
		shouldAutoAttach := false
		if agentConfig != nil {
			if tmuxParams, err := agentConfig.GetTmuxParams(); err == nil && tmuxParams != nil {
				shouldAutoAttach = tmuxParams.AutoAttach
			}
		}

		// Check if we can auto-attach (TTY available and autoAttach enabled)
		if shouldAutoAttach && term.IsTerminal(os.Stdin.Fd()) {
			ui.OutputLine("\nAuto-attaching to session...")
			tmuxCmd := exec.Command("tmux", "attach-session", "-t", info.TmuxSession)
			tmuxCmd.Stdin = os.Stdin
			tmuxCmd.Stdout = os.Stdout
			tmuxCmd.Stderr = os.Stderr
			return tmuxCmd.Run()
		}

		// Show manual attach instructions
		ui.OutputLine("\nTo attach to this session, run:")
		ui.OutputLine("  tmux attach-session -t %s", info.TmuxSession)
		attachID := sess.ID()
		if info.Index != "" {
			attachID = info.Index
		}
		ui.OutputLine("  or: amux attach %s", attachID)
	}

	return nil
}
