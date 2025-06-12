package agent

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

func attachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attach <session>",
		Short: "Attach to a running agent session",
		Long: `Attach to a running agent session.

This will connect you to the tmux session where the agent is running.
Use Ctrl-B D to detach from the session without stopping it.`,
		Args: cobra.ExactArgs(1),
		RunE: attachAgent,
	}
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

	// Create session manager
	sessionManager, err := createSessionManager(configManager, wsManager)
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
