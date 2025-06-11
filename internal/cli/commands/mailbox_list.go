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
	// Flags for list command
	listDirection string
)

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

	// Get mailbox path
	mailboxPath := mailboxManager.GetMailboxPath(sess.ID())

	// Check if mailbox exists
	if _, err := os.Stat(mailboxPath); os.IsNotExist(err) {
		ui.Info("No mailbox found for session %s", sessionID)
		return nil
	}

	// Count messages
	var inCount, outCount int
	var inMessages, outMessages []mailbox.Message

	// Get messages if we need to display them
	if listDirection == "" || listDirection == "in" {
		opts := mailbox.Options{
			SessionID: sess.ID(),
			Direction: mailbox.DirectionIn,
			Limit:     0, // Get all
		}
		inMessages, err = mailboxManager.ListMessages(opts)
		if err == nil {
			inCount = len(inMessages)
		}
	}

	if listDirection == "" || listDirection == "out" {
		opts := mailbox.Options{
			SessionID: sess.ID(),
			Direction: mailbox.DirectionOut,
			Limit:     0, // Get all
		}
		outMessages, err = mailboxManager.ListMessages(opts)
		if err == nil {
			outCount = len(outMessages)
		}
	}

	// Display header
	totalCount := inCount + outCount
	ui.PrintSectionHeader("📬", fmt.Sprintf("Mailbox for session %s", sessionID), totalCount)

	// Display summary
	fmt.Printf("Location: %s\n", mailboxPath)
	fmt.Printf("Messages: %d total (%d incoming, %d outgoing)\n\n", totalCount, inCount, outCount)

	// If filtering by direction, only show that section
	if listDirection == "in" && len(inMessages) > 0 {
		printMessageTable("📥 Incoming", inMessages)
	} else if listDirection == "out" && len(outMessages) > 0 {
		printMessageTable("📤 Outgoing", outMessages)
	} else if listDirection == "" {
		// Show both sections
		if len(inMessages) > 0 {
			printMessageTable("📥 Incoming", inMessages)
			if len(outMessages) > 0 {
				fmt.Println() // Add spacing between sections
			}
		}
		if len(outMessages) > 0 {
			printMessageTable("📤 Outgoing", outMessages)
		}
	}

	if totalCount == 0 || (listDirection != "" && inCount == 0 && outCount == 0) {
		ui.Info("No messages found")
	}

	return nil
}

func printMessageTable(title string, messages []mailbox.Message) {
	fmt.Printf("%s (%d)\n\n", title, len(messages))

	// Create table
	tbl := ui.NewTable("TIMESTAMP", "NAME", "SIZE", "AGE")

	// Add rows
	for _, msg := range messages {
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

		tbl.AddRow(timestamp, name, size, age)
	}

	tbl.Print()
}