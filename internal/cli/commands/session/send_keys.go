package session

import (
	"fmt"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/spf13/cobra"
)

var sendKeysCmd = &cobra.Command{
	Use:     "send-keys <session-id> <input>",
	Aliases: []string{"send"},
	Short:   "Send input to a running session",
	Long: `Send input text to a running session's stdin.

This command allows you to programmatically send commands or data to a session
that is running. The input is sent directly to the session's stdin through the
runtime interface (e.g., tmux send-keys).

Examples:
  # Send a simple command to a session
  amux session send-keys 1 "ls -la"

  # Send input using the session name
  amux session send-keys session-1 "npm test"

  # Send multi-line input (use quotes)
  amux session send-keys 2 "echo 'line 1'
echo 'line 2'"

Note: This command is only supported by certain runtimes (e.g., tmux).`,
	Args: cobra.ExactArgs(2),
	RunE: SendKeysToSession,
}

// SendKeysToSession implements the session send-keys command
func SendKeysToSession(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	sessionID := args[0]
	input := args[1]

	// Setup managers with project root detection
	_, sessionMgr, err := setupManagers()
	if err != nil {
		return err
	}

	// Send input to the session
	if err := sessionMgr.SendInput(ctx, sessionID, input); err != nil {
		return fmt.Errorf("failed to send input to session: %w", err)
	}

	ui.Success("Input sent to session %s", sessionID)

	return nil
}
