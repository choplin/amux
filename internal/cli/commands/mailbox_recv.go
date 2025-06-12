package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/mailbox"
	"github.com/aki/amux/internal/core/workspace"
)

// Flags for recv command
var recvQuiet bool

var mailboxRecvCmd = &cobra.Command{
	Use:   "recv <session>",
	Short: "Receive the latest message from an agent",
	Long: `Receive the latest message from an agent session.

Shows the most recent message in the 'out' directory (from the agent).

Examples:
  # Get latest message from agent
  amux mailbox recv s1
  amux mb recv s1
  # Get just the content (no metadata)
  amux mailbox recv s1 --quiet
  amux mb recv s1 -q`,
	Args: cobra.ExactArgs(1),
	RunE: recvFromSession,
}

func init() {
	// Add flags
	mailboxRecvCmd.Flags().BoolVarP(&recvQuiet, "quiet", "q", false, "Show only message content")
}

func recvFromSession(cmd *cobra.Command, args []string) error {
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

	// Get ID mapper
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
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

	// Get latest outgoing message
	opts := mailbox.Options{
		SessionID: sess.ID(),
		Direction: mailbox.DirectionOut,
		Limit:     1,
	}

	messages, err := mailboxManager.ListMessages(opts)
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	if len(messages) == 0 {
		ui.Info("No messages from agent in session %s", sessionID)
		return nil
	}

	// Read the message content
	content, err := mailboxManager.ReadMessage(messages[0])
	if err != nil {
		return fmt.Errorf("failed to read message: %w", err)
	}

	if recvQuiet {
		// Just print the content
		ui.Raw(content)
	} else {
		// Show metadata
		msg := messages[0]
		ui.PrintSectionHeader("📤", fmt.Sprintf("Latest from agent in session %s", sessionID), 1)
		ui.PrintKeyValue("Timestamp", msg.Timestamp.Format("2006-01-02 15:04:05"))
		ui.PrintKeyValue("Name", msg.Name)
		ui.PrintKeyValue("Age", ui.FormatTime(msg.Timestamp))
		ui.OutputLine("")
		ui.Raw(content)
		if !strings.HasSuffix(content, "\n") {
			ui.OutputLine("")
		}
	}

	return nil
}
