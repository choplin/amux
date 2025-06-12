package commands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/mailbox"
	"github.com/aki/amux/internal/core/workspace"
)

var (
	// Flags for show command
	showDirection string
	showTail      int
	showIn        bool
	showAll       bool
)

var mailboxShowCmd = &cobra.Command{
	Use:   "show <session> [index|latest]",
	Short: "Show messages from an agent session's mailbox",
	Long: `Show messages from an agent session's mailbox.

Can show a specific message by index, the latest message, or all messages.

Examples:
  # Show all messages (like old peek)
  amux mailbox show s1
  amux mailbox show s1 --all
  amux mb show s1
  # Show specific message by index
  amux mailbox show s1 3
  # Show latest outgoing message (from agent)
  amux mailbox show s1 latest
  # Show latest incoming message (to agent)
  amux mailbox show s1 latest --in
  # Show last 5 messages
  amux mailbox show s1 --tail 5`,
	Args: cobra.RangeArgs(1, 2),
	RunE: showMessages,
}

func init() {
	// Add flags
	mailboxShowCmd.Flags().StringVarP(&showDirection, "direction", "d", "", "Filter by direction (in/out)")
	mailboxShowCmd.Flags().IntVarP(&showTail, "tail", "t", 0, "Show last N messages")
	mailboxShowCmd.Flags().BoolVar(&showIn, "in", false, "When used with 'latest', show incoming instead of outgoing")
	mailboxShowCmd.Flags().BoolVarP(&showAll, "all", "a", false, "Show all messages")
}

func showMessages(cmd *cobra.Command, args []string) error {
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

	// Create mailbox manager
	mailboxManager := mailbox.NewManager(configManager.GetAmuxDir())

	// Determine what to show
	if len(args) > 1 {
		// Specific message requested
		selector := args[1]

		if selector == "latest" {
			// Show latest message
			direction := mailbox.DirectionOut
			if showIn {
				direction = mailbox.DirectionIn
			}
			return showLatestMessage(sessionID, sess.ID(), mailboxManager, direction)
		}

		// Try to parse as index
		index, err := strconv.Atoi(selector)
		if err != nil {
			return fmt.Errorf("invalid message selector: %s (expected number or 'latest')", selector)
		}
		return showMessageByIndex(sessionID, sess.ID(), mailboxManager, index)
	}

	// Default behavior - show all or tail
	if showTail > 0 {
		return showTailMessages(sessionID, sess.ID(), mailboxManager, showTail)
	}

	// Show all messages (old peek behavior)
	return showAllMessages(sessionID, sess.ID(), mailboxManager)
}

func showLatestMessage(sessionID, fullID string, mailboxManager *mailbox.Manager, direction mailbox.Direction) error {
	opts := mailbox.Options{
		SessionID: fullID,
		Direction: direction,
		Limit:     1,
	}

	messages, err := mailboxManager.ListMessages(opts)
	if err != nil {
		return fmt.Errorf("failed to list messages: %w", err)
	}

	if len(messages) == 0 {
		dirStr := "from agent"
		if direction == mailbox.DirectionIn {
			dirStr = "to agent"
		}
		ui.Info("No messages %s in session %s", dirStr, sessionID)
		return nil
	}

	// Read and display the message
	content, err := mailboxManager.ReadMessage(messages[0])
	if err != nil {
		return fmt.Errorf("failed to read message: %w", err)
	}

	msg := messages[0]
	icon := "📤"
	if direction == mailbox.DirectionIn {
		icon = "📥"
	}

	ui.PrintSectionHeader(icon, fmt.Sprintf("Latest message in session %s", sessionID), 1)
	ui.PrintKeyValue("Timestamp", msg.Timestamp.Format("2006-01-02 15:04:05"))
	ui.PrintKeyValue("Name", msg.Name)
	ui.PrintKeyValue("Direction", string(msg.Direction))
	ui.PrintKeyValue("Age", ui.FormatTime(msg.Timestamp))
	ui.OutputLine("")
	ui.Raw(content)
	if !strings.HasSuffix(content, "\n") {
		ui.OutputLine("")
	}

	return nil
}

func showMessageByIndex(sessionID, fullID string, mailboxManager *mailbox.Manager, index int) error {
	// Get all messages to find the one at index
	var allMessages []mailbox.Message

	// Get incoming messages
	inOpts := mailbox.Options{
		SessionID: fullID,
		Direction: mailbox.DirectionIn,
		Limit:     0,
	}
	inMessages, err := mailboxManager.ListMessages(inOpts)
	if err != nil {
		return fmt.Errorf("failed to list incoming messages: %w", err)
	}

	// Get outgoing messages
	outOpts := mailbox.Options{
		SessionID: fullID,
		Direction: mailbox.DirectionOut,
		Limit:     0,
	}
	outMessages, err := mailboxManager.ListMessages(outOpts)
	if err != nil {
		return fmt.Errorf("failed to list outgoing messages: %w", err)
	}

	// Combine and sort by timestamp (newest first)
	allMessages = append(allMessages, inMessages...)
	allMessages = append(allMessages, outMessages...)

	// Sort by timestamp descending
	sort.Slice(allMessages, func(i, j int) bool {
		return allMessages[i].Timestamp.After(allMessages[j].Timestamp)
	})

	// Check index bounds (1-based for user friendliness)
	if index < 1 || index > len(allMessages) {
		return fmt.Errorf("invalid index %d: session has %d messages", index, len(allMessages))
	}

	// Get the message (convert to 0-based)
	msg := allMessages[index-1]

	// Read and display
	content, err := mailboxManager.ReadMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to read message: %w", err)
	}

	icon := "📤"
	if msg.Direction == mailbox.DirectionIn {
		icon = "📥"
	}

	ui.PrintSectionHeader(icon, fmt.Sprintf("Message #%d in session %s", index, sessionID), 1)
	ui.PrintKeyValue("Timestamp", msg.Timestamp.Format("2006-01-02 15:04:05"))
	ui.PrintKeyValue("Name", msg.Name)
	ui.PrintKeyValue("Direction", string(msg.Direction))
	ui.PrintKeyValue("Age", ui.FormatTime(msg.Timestamp))
	ui.OutputLine("")
	ui.Raw(content)
	if !strings.HasSuffix(content, "\n") {
		ui.OutputLine("")
	}

	return nil
}

func showTailMessages(sessionID, fullID string, mailboxManager *mailbox.Manager, count int) error {
	// Get all messages
	opts := mailbox.Options{
		SessionID: fullID,
		Limit:     count,
	}

	if showDirection != "" {
		opts.Direction = mailbox.Direction(showDirection)
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
	ui.PrintSectionHeader("📬", fmt.Sprintf("Last %d messages for session %s", len(messages), sessionID), len(messages))

	for i, msg := range messages {
		// Format direction arrow
		dirArrow := "→" // Incoming
		if msg.Direction == mailbox.DirectionOut {
			dirArrow = "←" // Outgoing
		}

		// Format message header
		ui.OutputLine("%s %s %s %s",
			ui.DimStyle.Render(msg.Timestamp.Format("2006-01-02 15:04:05")),
			dirArrow,
			ui.BoldStyle.Render(msg.Name),
			ui.DimStyle.Render(fmt.Sprintf("[%s]", msg.Direction)),
		)

		// Read and show content
		content, err := mailboxManager.ReadMessage(msg)
		if err != nil {
			ui.OutputLine("  %s", ui.ErrorStyle.Render(fmt.Sprintf("Error reading message: %v", err)))
		} else {
			// Indent content
			lines := strings.Split(strings.TrimSpace(content), "\n")
			for _, line := range lines {
				ui.OutputLine("  %s", line)
			}
		}

		// Add spacing between messages
		if i < len(messages)-1 {
			ui.OutputLine("")
		}
	}

	return nil
}

func showAllMessages(sessionID, fullID string, mailboxManager *mailbox.Manager) error {
	// This replicates the old peek behavior
	var messages []mailbox.Message

	// Get messages based on direction filter
	if showDirection == "" || showDirection == "in" {
		opts := mailbox.Options{
			SessionID: fullID,
			Direction: mailbox.DirectionIn,
			Limit:     0,
		}
		inMessages, err := mailboxManager.ListMessages(opts)
		if err == nil {
			messages = append(messages, inMessages...)
		}
	}

	if showDirection == "" || showDirection == "out" {
		opts := mailbox.Options{
			SessionID: fullID,
			Direction: mailbox.DirectionOut,
			Limit:     0,
		}
		outMessages, err := mailboxManager.ListMessages(opts)
		if err == nil {
			messages = append(messages, outMessages...)
		}
	}

	if len(messages) == 0 {
		ui.Info("No messages found in mailbox")
		return nil
	}

	// Sort by timestamp (newest first)
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.After(messages[j].Timestamp)
	})

	// Display messages
	ui.PrintSectionHeader("📬", fmt.Sprintf("Mailbox for session %s", sessionID), len(messages))

	// Group by direction for display
	var inMsgs, outMsgs []mailbox.Message
	for _, msg := range messages {
		if msg.Direction == mailbox.DirectionIn {
			inMsgs = append(inMsgs, msg)
		} else {
			outMsgs = append(outMsgs, msg)
		}
	}

	// Show incoming messages
	if len(inMsgs) > 0 && (showDirection == "" || showDirection == "in") {
		ui.OutputLine("")
		ui.OutputLine("📥 Incoming (%d)", len(inMsgs))
		ui.OutputLine("")
		for i, msg := range inMsgs {
			ui.OutputLine("%d. %s %s %s",
				i+1,
				ui.DimStyle.Render(msg.Timestamp.Format("2006-01-02 15:04:05")),
				ui.BoldStyle.Render(msg.Name),
				ui.DimStyle.Render(ui.FormatTime(msg.Timestamp)),
			)

			// Show content preview
			content, err := mailboxManager.ReadMessage(msg)
			if err == nil {
				lines := strings.Split(strings.TrimSpace(content), "\n")
				preview := lines[0]
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}
				ui.OutputLine("   %s", ui.DimStyle.Render(preview))
			}
			ui.OutputLine("")
		}
	}

	// Show outgoing messages
	if len(outMsgs) > 0 && (showDirection == "" || showDirection == "out") {
		if len(inMsgs) > 0 {
			ui.OutputLine("")
		}
		ui.OutputLine("📤 Outgoing (%d)", len(outMsgs))
		ui.OutputLine("")
		for i, msg := range outMsgs {
			ui.OutputLine("%d. %s %s %s",
				len(inMsgs)+i+1,
				ui.DimStyle.Render(msg.Timestamp.Format("2006-01-02 15:04:05")),
				ui.BoldStyle.Render(msg.Name),
				ui.DimStyle.Render(ui.FormatTime(msg.Timestamp)),
			)

			// Show content preview
			content, err := mailboxManager.ReadMessage(msg)
			if err == nil {
				lines := strings.Split(strings.TrimSpace(content), "\n")
				preview := lines[0]
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}
				ui.OutputLine("   %s", ui.DimStyle.Render(preview))
			}
			ui.OutputLine("")
		}
	}

	ui.OutputLine("Use 'amux mailbox show %s <index>' to read a specific message", sessionID)

	return nil
}
