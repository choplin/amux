package session

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/tail"
)

func logsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <session>",
		Short: "View session output",
		Long: `View the output from an agent session.

Shows the current content of the agent's terminal.
Use -f/--follow to continuously stream new output.`,
		Args: cobra.ExactArgs(1),
		RunE: viewSessionLogs,
	}

	// Logs command flags
	cmd.Flags().BoolVarP(&followLogs, "follow", "f", false, "Follow log output (tail -f behavior)")
	cmd.Flags().StringVar(&followInterval, "interval", "1s", "Refresh interval when following logs")

	return cmd
}

func viewSessionLogs(cmd *cobra.Command, args []string) error {
	sessionID := args[0]

	// Get session manager
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}
	sessionManager, err := session.SetupManager(projectRoot)
	if err != nil {
		return err
	}

	// Get session
	sess, err := sessionManager.ResolveSession(cmd.Context(), session.Identifier(sessionID))
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Check if we need to follow logs
	if followLogs {
		// Stream logs continuously
		return streamSessionLogs(sess)
	}

	// Type assert to TerminalSession
	terminalSess, ok := sess.(session.TerminalSession)
	if !ok {
		return fmt.Errorf("session does not support terminal operations")
	}

	// Get snapshot of output (0 = all lines for non-follow mode)
	output, err := terminalSess.GetOutput(0)
	if err != nil {
		return fmt.Errorf("failed to get session output: %w", err)
	}

	// Print output
	ui.Raw(string(output))
	return nil
}

// tailSessionLogs is a wrapper for following session logs
func tailSessionLogs(cmd *cobra.Command, args []string) error {
	// Set follow flag to true
	followLogs = true
	// Reuse viewSessionLogs logic
	return viewSessionLogs(cmd, args)
}

// streamSessionLogs continuously streams session output
func streamSessionLogs(sess session.Session) error {
	// Parse refresh interval
	interval, err := time.ParseDuration(followInterval)
	if err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}

	// Set up signal handling for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals in a goroutine
	go func() {
		<-sigChan
		cancel()
	}()

	ui.Output("Following logs (refresh every %s, press Ctrl+C to stop)", interval)

	// Create tail options with custom interval
	opts := tail.DefaultOptions()
	opts.PollInterval = interval
	opts.Writer = os.Stdout

	// Use the tail package to follow logs
	tailer := tail.New(sess, opts)
	if tailer == nil {
		return fmt.Errorf("session does not support terminal operations")
	}
	err = tailer.Follow(ctx)

	if err == context.Canceled {
		ui.OutputLine("\nStopped following logs")
		return nil
	}

	if err == nil {
		ui.OutputLine("\nSession ended")
		return nil
	}

	return err
}
