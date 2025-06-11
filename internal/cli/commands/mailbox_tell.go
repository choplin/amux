package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/common"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/mailbox"
	"github.com/aki/amux/internal/core/workspace"
)

var (
	// Flags for tell command
	tellFile string
)

var mailboxTellCmd = &cobra.Command{
	Use:   "tell <session> <message>",
	Short: "Send a message to an agent session",
	Long: `Send a message to an agent session's mailbox.

The message can be provided as command arguments or from a file.

Examples:
  # Send a simple message
  amux mailbox tell s1 "Please focus on the authentication module"
  amux mb tell s1 "Please focus on the authentication module"

  # Send a message from a file
  amux mailbox tell s1 --file requirements.md`,
	Args: cobra.MinimumNArgs(1),
	RunE: tellSession,
}

func init() {
	// Add flags
	mailboxTellCmd.Flags().StringVarP(&tellFile, "file", "f", "", "Read message from file")
}

func tellSession(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	// Determine message content
	var content string
	var name string

	if tellFile != "" {
		// Read from file
		data, err := os.ReadFile(tellFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		content = string(data)
		name = "file-" + tellFile
	} else {
		// Require message as argument
		if len(args) < 2 {
			return fmt.Errorf("message required when not using --file")
		}
		// Join all remaining arguments as the message
		content = strings.Join(args[1:], " ")
		// Generate name from first few words
		words := strings.Fields(content)
		if len(words) > 3 {
			name = strings.Join(words[:3], "-")
		} else {
			name = strings.Join(words, "-")
		}
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

	// Create mailbox manager
	mailboxManager := mailbox.NewManager(configManager.GetAmuxDir())

	// Send message
	if err := mailboxManager.SendMessage(sess.ID(), name, content); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	ui.Success("Message sent to session %s", sessionID)
	return nil
}
