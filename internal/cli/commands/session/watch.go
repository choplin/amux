package session

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch <session-id>",
	Short: "Watch real-time output from a session",
	Long:  `Watch real-time output from a session by connecting to its output socket.`,
	Args:  cobra.ExactArgs(1),
	RunE:  WatchSession,
}

// WatchSession implements the session watch command
func WatchSession(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	// Get managers
	_, sessionMgr, err := setupManagers()
	if err != nil {
		return err
	}

	// Get session info to find socket path
	ctx := cmd.Context()
	sess, err := sessionMgr.Get(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Get socket path from session info
	socketPath := sess.SocketPath
	if socketPath == "" {
		// Fallback for old sessions without socket path
		tmpDir := os.Getenv("TMPDIR")
		if tmpDir == "" {
			tmpDir = "/tmp"
		}
		socketPath = filepath.Join(tmpDir, fmt.Sprintf("amux-%s.sock", sessionID))
	}

	// Connect to socket
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to session output: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Copy output to stdout
	_, err = io.Copy(os.Stdout, conn)
	if err != nil && err != io.EOF {
		return fmt.Errorf("error reading output: %w", err)
	}

	return nil
}
