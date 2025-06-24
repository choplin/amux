package session

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/adapters/tmux"
	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/terminal"
	"github.com/aki/amux/internal/core/workspace"
)

func attachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attach <session>",
		Short: "Attach to a running session",
		Long: `Attach to a running agent session.

This will connect you to the tmux session where the agent is running.
Use Ctrl-B D to detach from the session without stopping it.`,
		Args: cobra.ExactArgs(1),
		RunE: attachSession,
	}
}

func attachSession(cmd *cobra.Command, args []string) error {
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
	sess, err := sessionManager.ResolveSession(cmd.Context(), session.Identifier(sessionID))
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Check if running
	if !sess.Status().IsRunning() {
		return fmt.Errorf("session is not running (status: %s)", sess.Status())
	}

	// Get tmux session name
	info := sess.Info()
	if info.TmuxSession == "" {
		return fmt.Errorf("session does not have a tmux session")
	}

	// Detect current terminal size and resize tmux window
	width, height := terminal.GetSize()
	tmuxAdapter, err := tmux.NewAdapter()
	if err == nil {
		if err := tmuxAdapter.ResizeWindow(info.TmuxSession, width, height); err != nil {
			// Log warning but don't fail - resize is not critical
			slog.Warn("failed to resize tmux window", "error", err, "session", info.TmuxSession)
		}
	}

	// Execute tmux attach
	ui.OutputLine("Attaching to session %s...", sessionID)
	tmuxCmd := exec.Command("tmux", "attach-session", "-t", info.TmuxSession)
	tmuxCmd.Stdin = os.Stdin
	tmuxCmd.Stdout = os.Stdout
	tmuxCmd.Stderr = os.Stderr

	return tmuxCmd.Run()
}
