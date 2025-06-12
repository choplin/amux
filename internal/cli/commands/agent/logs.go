package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/tail"
	"github.com/aki/amux/internal/core/workspace"
)

func logsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs <session>",
		Short: "View agent session output",
		Long: `View the output from an agent session.

Shows the current content of the agent's terminal.
Use -f/--follow to continuously stream new output.`,
		Args: cobra.ExactArgs(1),
		RunE: viewAgentLogs,
	}

	// Logs command flags
	cmd.Flags().BoolVarP(&followLogs, "follow", "f", false, "Follow log output (tail -f behavior)")

	return cmd
}

func viewAgentLogs(cmd *cobra.Command, args []string) error {
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

	// Check if we need to follow logs
	if followLogs {
		// Stream logs continuously
		return streamSessionLogs(sess)
	}

	// Get snapshot of output
	output, err := sess.GetOutput()
	if err != nil {
		return fmt.Errorf("failed to get session output: %w", err)
	}

	// Print output
	ui.Raw(string(output))
	return nil
}

// tailAgentLogs is a wrapper for following agent logs
func tailAgentLogs(cmd *cobra.Command, args []string) error {
	// Set follow flag to true
	followLogs = true
	// Reuse viewAgentLogs logic
	return viewAgentLogs(cmd, args)
}

// streamSessionLogs continuously streams session output
func streamSessionLogs(sess session.Session) error {
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

	ui.Info("Following logs... (press Ctrl+C to stop)")

	// Use the tail package to follow logs
	err := tail.FollowFunc(ctx, sess, os.Stdout)

	if err == context.Canceled {
		ui.Info("\nStopped following logs")
		return nil
	}

	if err == nil {
		ui.Info("\nSession ended")
		return nil
	}

	return err
}
