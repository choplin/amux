package session

import (
	"fmt"
	"io"
	"os"

	"github.com/aki/amux/internal/config"
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

	// Get working directory
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Create config manager
	configMgr := config.NewManager(wd)
	if !configMgr.IsInitialized() {
		return fmt.Errorf("amux not initialized. Run 'amux init' first")
	}

	// Get session manager
	sessionMgr := getSessionManager(configMgr)

	// Get logs
	reader, err := sessionMgr.Logs(ctx, sessionID, logsOpts.follow)
	if err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	// Copy logs to stdout
	if _, err := io.Copy(os.Stdout, reader); err != nil {
		return fmt.Errorf("failed to read logs: %w", err)
	}

	return nil
}
