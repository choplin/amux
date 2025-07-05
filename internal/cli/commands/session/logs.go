package session

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <session-id>",
	Short: "Show logs from a session",
	Long:  `Show logs from a session.`,
	Args:  cobra.ExactArgs(1),
	RunE:  ShowLogs,
}

var logsOpts struct {
	follow bool
	tail   int
}

func init() {
	logsCmd.Flags().BoolVarP(&logsOpts.follow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&logsOpts.tail, "tail", "n", 0, "Number of lines to show from the end")
}

// BindLogsFlags binds command flags to logsOpts
func BindLogsFlags(cmd *cobra.Command) {
	logsOpts.follow, _ = cmd.Flags().GetBool("follow")
	logsOpts.tail, _ = cmd.Flags().GetInt("tail")
}

// SetLogsFollow sets the follow flag
func SetLogsFollow(follow bool) {
	logsOpts.follow = follow
}

// ShowLogs implements the session logs command
func ShowLogs(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	sessionID := args[0]

	// Setup managers with project root detection
	_, sessionMgr, err := setupManagers()
	if err != nil {
		return err
	}

	// Get logs
	reader, err := sessionMgr.Logs(ctx, sessionID, logsOpts.follow)
	if err != nil {
		// Check if session not found
		if _, getErr := sessionMgr.Get(ctx, sessionID); getErr != nil {
			return fmt.Errorf("session '%s' not found. Run 'amux ps' to see active sessions", sessionID)
		}
		return fmt.Errorf("failed to get logs for session '%s': %w", sessionID, err)
	}

	// Check if reader is nil
	if reader == nil {
		return fmt.Errorf("no logs available for session '%s'", sessionID)
	}
	defer func() {
		_ = reader.Close()
	}()

	// Copy logs to stdout
	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return fmt.Errorf("failed to read logs: %w", err)
	}

	return nil
}
