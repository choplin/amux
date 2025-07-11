package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/runtime/proxy"
)

// NewProxyCommand creates the proxy command
func NewProxyCommand() *cobra.Command {
	var (
		statusPath string
		logPath    string
		socketPath string
		sessionDir string
		foreground bool
	)

	cmd := &cobra.Command{
		Use:    "proxy",
		Short:  "Internal command to proxy process I/O and monitor status",
		Hidden: true, // This is an internal command
		Args:   cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// All paths must be provided by the runtime
			if statusPath == "" {
				return fmt.Errorf("--status-path is required")
			}
			if socketPath == "" {
				return fmt.Errorf("--socket-path is required")
			}
			if sessionDir == "" {
				return fmt.Errorf("--session-dir is required")
			}

			opts := proxy.Options{
				SessionDir: sessionDir,
				StatusPath: statusPath,
				LogPath:    logPath,
				SocketPath: socketPath,
				Command:    args,
				Foreground: foreground,
			}

			p, err := proxy.New(opts)
			if err != nil {
				return err
			}

			return p.Run()
		},
	}

	cmd.Flags().StringVar(&statusPath, "status-path", "", "Path to status file")
	cmd.Flags().StringVar(&logPath, "log-path", "", "Path to log file (optional)")
	cmd.Flags().StringVar(&socketPath, "socket-path", "", "Path to Unix socket for output streaming")
	cmd.Flags().StringVar(&sessionDir, "session-dir", "", "Session directory for storing run data")
	cmd.Flags().BoolVar(&foreground, "foreground", false, "Run in foreground mode (direct I/O)")
	_ = cmd.MarkFlagRequired("status-path")
	_ = cmd.MarkFlagRequired("socket-path")
	_ = cmd.MarkFlagRequired("session-dir")

	return cmd
}
