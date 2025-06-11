package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/context"
	"github.com/aki/amux/internal/core/workspace"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage working context files",
	Long: `Manage working context files for AI agents.

Working context files help AI agents maintain context and track progress:
- background.md: Task requirements and constraints
- plan.md: Implementation approach
- working-log.md: Progress tracking
- results-summary.md: Final outcomes`,
}

var contextShowCmd = &cobra.Command{
	Use:   "show [workspace]",
	Short: "Show context file paths",
	Long: `Show the paths to working context files in a workspace.

If no workspace is specified, uses the most recent workspace.`,
	RunE: showContext,
}

var contextInitCmd = &cobra.Command{
	Use:   "init [workspace]",
	Short: "Initialize context files",
	Long: `Initialize working context files in a workspace.

Creates template files if they don't already exist.`,
	RunE: initContext,
}

func init() {
	workspaceCmd.AddCommand(contextCmd)
	contextCmd.AddCommand(contextShowCmd)
	contextCmd.AddCommand(contextInitCmd)
}

func showContext(cmd *cobra.Command, args []string) error {
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

	// Get workspace
	var ws *workspace.Workspace
	if len(args) > 0 {
		ws, err = wsManager.ResolveWorkspace(args[0])
		if err != nil {
			return fmt.Errorf("failed to resolve workspace: %w", err)
		}
	} else {
		// Get the most recent workspace
		workspaces, err := wsManager.List(workspace.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list workspaces: %w", err)
		}
		if len(workspaces) == 0 {
			return fmt.Errorf("no workspaces available")
		}
		ws = workspaces[0]
	}

	// Create context manager
	contextManager := context.NewManager(ws.Path)

	// Show context information
	ui.Info("Workspace: %s (%s)", ws.Name, ws.ID)
	ui.Info("Context directory: %s", contextManager.GetContextPath())
	fmt.Println()

	// Check if context exists
	if !contextManager.Exists() {
		ui.Warning("Context not initialized. Run 'amux ws context init' to create template files.")
		return nil
	}

	// List context files
	ui.Info("Context files:")
	files := []struct {
		name string
		desc string
	}{
		{context.BackgroundFile, "Task requirements and constraints"},
		{context.PlanFile, "Implementation approach"},
		{context.WorkingLogFile, "Progress tracking"},
		{context.ResultsSummaryFile, "Final outcomes"},
	}

	for _, file := range files {
		path := contextManager.GetFilePath(file.name)
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("  • %s\n", path)
			fmt.Printf("    %s\n", ui.DimStyle.Render(file.desc))
		}
	}

	// Show environment variable hint
	fmt.Println()
	ui.Info("Environment variable available in sessions:")
	fmt.Printf("  AMUX_CONTEXT_PATH=%s\n", contextManager.GetContextPath())

	return nil
}

func initContext(cmd *cobra.Command, args []string) error {
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

	// Get workspace
	var ws *workspace.Workspace
	if len(args) > 0 {
		ws, err = wsManager.ResolveWorkspace(args[0])
		if err != nil {
			return fmt.Errorf("failed to resolve workspace: %w", err)
		}
	} else {
		// Get the most recent workspace
		workspaces, err := wsManager.List(workspace.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list workspaces: %w", err)
		}
		if len(workspaces) == 0 {
			return fmt.Errorf("no workspaces available")
		}
		ws = workspaces[0]
	}

	// Create context manager
	contextManager := context.NewManager(ws.Path)

	// Initialize context
	if err := contextManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize context: %w", err)
	}

	ui.Success("Initialized working context in workspace '%s'", ws.Name)
	ui.Info("Context directory: %s", contextManager.GetContextPath())

	return nil
}
