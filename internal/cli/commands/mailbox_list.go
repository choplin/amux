package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/mailbox"
	"github.com/aki/amux/internal/core/workspace"
)

// Flags for list command
var listDirection string

var mailboxListCmd = &cobra.Command{
	Use:   "list <session>",
	Short: "List files in an agent session's mailbox",
	Long: `List files in an agent session's mailbox.

Shows message files in the mailbox directory with their metadata.

Examples:
  # List all messages
  amux mailbox list s1
  amux mb list s1

  # List only incoming messages
  amux mailbox list s1 --direction in

  # List only outgoing messages
  amux mailbox list s1 --direction out`,
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"ls"},
	RunE:    listMailbox,
}

func init() {
	// Add flags
	mailboxListCmd.Flags().StringVarP(&listDirection, "direction", "d", "", "Filter by direction (in/out)")
}

func listMailbox(cmd *cobra.Command, args []string) error {
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
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return fmt.Errorf("failed to create ID mapper: %w", err)
	}

	// Create session manager
	log := CreateLogger()
	sessionManager, err := createSessionManager(configManager, wsManager, idMapper, log)
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

	// Get mailbox path
	mailboxPath := mailboxManager.GetMailboxPath(sess.ID())

	// Check if mailbox exists
	if _, err := os.Stat(mailboxPath); os.IsNotExist(err) {
		ui.Info("No mailbox found for session %s", sessionID)
		return nil
	}

	// Get all messages
	var allMessages []mailbox.Message
	var inCount, outCount int

	// Get incoming messages
	if listDirection == "" || listDirection == "in" {
		opts := mailbox.Options{
			SessionID: sess.ID(),
			Direction: mailbox.DirectionIn,
			Limit:     0,
		}
		inMessages, err := mailboxManager.ListMessages(opts)
		if err == nil {
			inCount = len(inMessages)
			allMessages = append(allMessages, inMessages...)
		}
	}

	// Get outgoing messages
	if listDirection == "" || listDirection == "out" {
		opts := mailbox.Options{
			SessionID: sess.ID(),
			Direction: mailbox.DirectionOut,
			Limit:     0,
		}
		outMessages, err := mailboxManager.ListMessages(opts)
		if err == nil {
			outCount = len(outMessages)
			allMessages = append(allMessages, outMessages...)
		}
	}

	// Sort all messages by timestamp (newest first)
	sort.Slice(allMessages, func(i, j int) bool {
		return allMessages[i].Timestamp.After(allMessages[j].Timestamp)
	})

	// Display header
	totalCount := inCount + outCount
	ui.PrintSectionHeader("📬", fmt.Sprintf("Mailbox for session %s", sessionID), totalCount)

	// Display summary
	ui.PrintKeyValue("Location", mailboxPath)
	ui.OutputLine("Messages: %d total (%d incoming, %d outgoing)\n", totalCount, inCount, outCount)

	// Display messages in a single table with global indices
	if len(allMessages) > 0 {
		printMessageTableWithIndex(allMessages)
		ui.OutputLine("\nUse 'amux mailbox show %s <#>' to read a specific message", sessionID)
	}

	if totalCount == 0 || (listDirection != "" && inCount == 0 && outCount == 0) {
		ui.Info("No messages found")
	}

	return nil
}

func printMessageTableWithIndex(messages []mailbox.Message) {
	// Create table with index column
	tbl := ui.NewTable("#", "DIR", "TIMESTAMP", "NAME", "SIZE", "AGE")

	// Add rows with indices
	for i, msg := range messages {
		// Get file info for size
		fileInfo, err := os.Stat(msg.Path)
		size := "-"
		if err == nil {
			size = ui.FormatSize(fileInfo.Size())
		}

		// Format timestamp
		timestamp := msg.Timestamp.Format("2006-01-02 15:04:05")

		// Clean up name for display (remove .md extension)
		name := strings.TrimSuffix(msg.Name, ".md")

		// Format age
		age := ui.FormatTime(msg.Timestamp)

		// Direction indicator
		dir := "→" // incoming
		if msg.Direction == mailbox.DirectionOut {
			dir = "←" // outgoing
		}

		// Index is 1-based for user friendliness
		tbl.AddRow(i+1, dir, timestamp, name, size, age)
	}

	tbl.Print()
}
