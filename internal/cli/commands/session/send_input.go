package session

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
)

// sendInputCmd returns the 'session send-input' command
func sendInputCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "send-input <session-id> <input-text>",
		Aliases: []string{"send"},
		Short:   "Send input to a running session",
		Long: `Send input text to a running AI agent session's stdin.

This command allows you to programmatically send commands or data to an AI agent
that is running in a session. The input is sent directly to the session's stdin
through the tmux interface.

Examples:
  # Send a simple command to a session
  amux session send-input 1 "ls -la"

  # Send input using the session name
  amux session send-input feat-auth-agent "npm test"

  # Send multi-line input (use quotes)
  amux session send-input 2 "echo 'line 1'
echo 'line 2'"`,
		Args: cobra.ExactArgs(2),
		RunE: sendInputToSession,
	}

	return cmd
}

// sendInputToSession executes the send-input command
func sendInputToSession(cmd *cobra.Command, args []string) error {
	sessionID := args[0]
	inputText := args[1]

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to find project root: %w", err)
	}

	// Create managers
	configManager := config.NewManager(projectRoot)

	// Create both managers together with proper initialization
	_, sessionManager, err := createManagers(configManager)
	if err != nil {
		return err
	}

	// Get the session
	sess, err := sessionManager.ResolveSession(cmd.Context(), session.Identifier(sessionID))
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Check if session is running
	if !sess.Status().IsRunning() {
		return fmt.Errorf("session %s is not running (current status: %s)", sessionID, sess.Status())
	}

	// Type assert to TerminalSession
	terminalSess, ok := sess.(session.TerminalSession)
	if !ok {
		return fmt.Errorf("session does not support terminal operations")
	}

	// Send input to the session
	if err := terminalSess.SendInput(inputText); err != nil {
		return fmt.Errorf("failed to send input to session: %w", err)
	}

	// Display success message
	ui.OutputLine("Input sent to session %s", sessionID)

	return nil
}
