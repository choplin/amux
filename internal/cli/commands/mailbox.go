package commands

import (
	"github.com/spf13/cobra"
)

var mailboxCmd = &cobra.Command{
	Use:     "mailbox",
	Aliases: []string{"mb"},
	Short:   "Manage agent session mailboxes",
	Long: `Manage mailboxes for agent sessions.

Mailboxes provide asynchronous communication with running agents.
Each session has a mailbox directory with incoming and outgoing messages.`,
}

func init() {
	// Add subcommands
	mailboxCmd.AddCommand(mailboxListCmd)
	mailboxCmd.AddCommand(mailboxTellCmd)
	mailboxCmd.AddCommand(mailboxPeekCmd)
}
