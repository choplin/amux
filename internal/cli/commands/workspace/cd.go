package workspace

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/config"
	"github.com/aki/amux/internal/workspace"
)

var cdWorkspaceCmd = &cobra.Command{
	Use:   "cd <workspace-name-or-id>",
	Short: "Open a subshell in the workspace directory",
	Long: `Open a new shell in the workspace directory. Exit the shell to return to the original directory.

Examples:
  # Enter workspace by ID
  amux ws cd 1

  # Enter workspace by name
  amux ws cd feat-auth

  # Exit the workspace (in the subshell)
  exit`,
	Args: cobra.ExactArgs(1),
	RunE: runCdWorkspace,
}

func runCdWorkspace(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}
	manager, err := workspace.SetupManager(projectRoot)
	if err != nil {
		return err
	}

	// Resolve workspace by name or ID
	ws, err := manager.ResolveWorkspace(cmd.Context(), workspace.Identifier(identifier))
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Get user's shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	// Create a new shell process
	shellCmd := exec.Command(shell)
	shellCmd.Dir = ws.Path
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	// Set environment variable to indicate we're in an amux workspace
	shellCmd.Env = append(os.Environ(),
		fmt.Sprintf("AMUX_WORKSPACE=%s", ws.Name),
		fmt.Sprintf("AMUX_WORKSPACE_ID=%s", ws.ID),
		fmt.Sprintf("AMUX_WORKSPACE_PATH=%s", ws.Path),
	)

	// Print information about entering the workspace
	ui.OutputLine("Entering workspace: %s", ws.Name)
	ui.PrintKeyValue("Path", ws.Path)
	ui.OutputLine("\nExit the shell to return to your original directory")

	// Run the shell
	if err := shellCmd.Run(); err != nil {
		// Don't treat exit as an error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() > 0 {
			// User exited with non-zero code, this is fine
			return nil
		}
		return fmt.Errorf("failed to run shell: %w", err)
	}

	return nil
}
