package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/common"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/mailbox"
	"github.com/aki/amux/internal/core/workspace"
)

var (
	// Flags for peek command
	peekDirection string
	peekLimit     int
	peekVerbose   bool
)

var mailboxPeekCmd = &cobra.Command{
	Use:   "peek <session>",
	Short: "View messages in an agent session's mailbox",
	Long: `View messages in an agent session's mailbox.

Shows messages in both directions (to and from the agent) by default.

Examples:
  # View all messages
  amux mailbox peek s1
  amux mb peek s1

  # View only incoming messages (to the agent)
  amux mailbox peek s1 --direction in

  # View only outgoing messages (from the agent)
  amux mailbox peek s1 --direction out

  # View last 5 messages with full content
  amux mailbox peek s1 --limit 5 --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: peekSession,
}

func init() {
	// Add flags
	mailboxPeekCmd.Flags().StringVarP(&peekDirection, "direction", "d", "", "Filter by direction (in/out)")
	mailboxPeekCmd.Flags().IntVarP(&peekLimit, "limit", "l", 10, "Limit number of messages")
	mailboxPeekCmd.Flags().BoolVarP(&peekVerbose, "verbose", "v", false, "Show full message content")
}

func peekSession(cmd *cobra.Command, args []string) error {
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

	// Create mailbox manager
	mailboxManager := mailbox.NewManager(configManager.GetAmuxDir())

	// List messages
	opts := mailbox.Options{
		SessionID: sess.ID(),
		Limit:     peekLimit,
	}

	if peekDirection != "" {
		opts.Direction = mailbox.Direction(peekDirection)
	}

	messages, err := mailboxManager.ListMessages(opts)
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	if len(messages) == 0 {
		ui.Info("No messages found in mailbox")
		return nil
	}

	// Display messages
	ui.PrintSectionHeader("📬", fmt.Sprintf("Mailbox for session %s", sessionID), len(messages))

	for i, msg := range messages {
		// Format direction arrow
		dirArrow := "→" // Incoming
		if msg.Direction == mailbox.DirectionOut {
			dirArrow = "←" // Outgoing
		}

		// Format message header
		fmt.Printf("%s %s %s %s\n",
			ui.DimStyle.Render(msg.Timestamp.Format("2006-01-02 15:04:05")),
			dirArrow,
			ui.BoldStyle.Render(msg.Name),
			ui.DimStyle.Render(fmt.Sprintf("[%s]", msg.Direction)),
		)

		// Show content if verbose
		if peekVerbose {
			content, err := mailboxManager.ReadMessage(msg)
			if err != nil {
				fmt.Printf("  %s\n", ui.ErrorStyle.Render(fmt.Sprintf("Error reading message: %v", err)))
			} else {
				// Indent content
				lines := strings.Split(strings.TrimSpace(content), "\n")
				for _, line := range lines {
					fmt.Printf("  %s\n", line)
				}
			}
		}

		// Add spacing between messages
		if i < len(messages)-1 {
			fmt.Println()
		}
	}

	return nil
}
