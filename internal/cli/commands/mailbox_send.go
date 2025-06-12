package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/mailbox"
	"github.com/aki/amux/internal/core/workspace"
)

// Flags for send command
var sendFile string

var mailboxSendCmd = &cobra.Command{
	Use:   "send <session> [message]",
	Short: "Send a message to an agent session",
	Long: `Send a message to an agent session's mailbox.

The message can be provided as:
- Command line arguments
- From a file with -f/--file
- From stdin (when no message argument is provided)

Examples:
  # Send a simple message
  amux mailbox send s1 "Please focus on the authentication module"
  amux mb send s1 "Please focus on the authentication module"
  # Send from a file
  amux mailbox send s1 --file requirements.md
  # Send from stdin
  echo "Update the tests" | amux mailbox send s1
  amux mailbox send s1 < requirements.md`,
	Args: cobra.MinimumNArgs(1),
	RunE: sendToSession,
}

func init() {
	// Add flags
	mailboxSendCmd.Flags().StringVarP(&sendFile, "file", "f", "", "Read message from file")
}

func sendToSession(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	// Determine message content
	var content string
	var name string

	if sendFile != "" {
		// Read from file
		data, err := os.ReadFile(sendFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		content = string(data)
		name = "file-" + sendFile
	} else if len(args) > 1 {
		// Message provided as arguments
		content = strings.Join(args[1:], " ")
		// Generate name from first few words
		words := strings.Fields(content)
		if len(words) > 3 {
			name = strings.Join(words[:3], "-")
		} else {
			name = strings.Join(words, "-")
		}
	} else {
		// Read from stdin
		stat, err := os.Stdin.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat stdin: %w", err)
		}

		// Check if stdin is a pipe or redirect
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return fmt.Errorf("no message provided: use arguments, --file, or pipe input")
		}

		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		content = string(data)
		name = "stdin-message"
	}

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

	// Create mailbox manager
	mailboxManager := mailbox.NewManager(configManager.GetAmuxDir())

	// Send message
	if err := mailboxManager.SendMessage(sess.ID(), name, content); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	ui.Success("Message sent to session %s", sessionID)
	return nil
}
