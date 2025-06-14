package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/aki/amux/internal/cli/ui"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/hooks"
	"github.com/aki/amux/internal/core/workspace"
)

var workspaceCmd = &cobra.Command{
	Use: "workspace",

	Aliases: []string{"ws"},

	Short: "Manage Amux workspaces",

	Long: "Create, list, and remove isolated development workspaces",
}

var (

	// Create flags

	createBaseBranch string

	createBranch string

	createAgentID string

	createDescription string

	createNoHooks bool

	// List flags

	listOneline bool

	// Prune flags

	pruneDays int

	pruneDryRun bool

	// Remove flags

	removeForce bool

	removeNoHooks bool
)

func init() {
	// Add subcommands

	workspaceCmd.AddCommand(createWorkspaceCmd)

	workspaceCmd.AddCommand(listWorkspaceCmd)

	workspaceCmd.AddCommand(showWorkspaceCmd)

	workspaceCmd.AddCommand(removeWorkspaceCmd)

	workspaceCmd.AddCommand(pruneWorkspaceCmd)

	workspaceCmd.AddCommand(cdWorkspaceCmd)

	// Create command flags

	createWorkspaceCmd.Flags().StringVarP(&createBaseBranch, "base-branch", "b", "", "Base branch to create workspace from")

	createWorkspaceCmd.Flags().StringVar(&createBranch, "branch", "", "Use existing branch instead of creating new one")

	createWorkspaceCmd.Flags().StringVarP(&createAgentID, "agent", "a", "", "Agent ID to assign to workspace")

	createWorkspaceCmd.Flags().StringVarP(&createDescription, "description", "d", "", "Description of the workspace")

	createWorkspaceCmd.Flags().BoolVar(&createNoHooks, "no-hooks", false, "Skip running hooks for this operation")

	// List command flags

	listWorkspaceCmd.Flags().BoolVar(&listOneline, "oneline", false, "Show one workspace per line (for use with fzf)")

	// Prune command flags

	pruneWorkspaceCmd.Flags().IntVarP(&pruneDays, "days", "d", 7, "Remove workspaces idle for more than N days")

	pruneWorkspaceCmd.Flags().BoolVar(&pruneDryRun, "dry-run", false, "Show what would be removed without removing")

	// Remove command flags

	removeWorkspaceCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Force removal without confirmation")

	removeWorkspaceCmd.Flags().BoolVar(&removeNoHooks, "no-hooks", false, "Skip running hooks for this operation")
}

var createWorkspaceCmd = &cobra.Command{
	Use: "create <name>",

	Short: "Create a new workspace",

	Args: cobra.ExactArgs(1),

	RunE: runCreateWorkspace,
}

var listWorkspaceCmd = &cobra.Command{
	Use: "list",

	Aliases: []string{"ls"},

	Short: "List all workspaces",

	Long: `List all workspaces in the project.

Examples:
  # List workspaces with detailed view
  amux ws ls

  # List workspaces in oneline format for scripting
  amux ws ls --oneline

  # Use with fzf to select a workspace
  amux ws ls --oneline | fzf | cut -f1

  # Remove selected workspace with fzf
  amux ws rm $(amux ws ls --oneline | fzf | cut -f1)`,

	RunE: runListWorkspace,
}

var showWorkspaceCmd = &cobra.Command{
	Use:   "show <workspace-name-or-id>",
	Short: "Show detailed information about a workspace",
	Args:  cobra.ExactArgs(1),
	RunE:  runShowWorkspace,
}

var removeWorkspaceCmd = &cobra.Command{
	Use: "remove <workspace-name-or-id>",

	Aliases: []string{"rm"},

	Short: "Remove a workspace by name or ID",

	Args: cobra.ExactArgs(1),

	RunE: runRemoveWorkspace,
}

var pruneWorkspaceCmd = &cobra.Command{
	Use: "prune",

	Short: "Remove old idle workspaces",

	RunE: runPruneWorkspace,
}

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

func runCreateWorkspace(cmd *cobra.Command, args []string) error {
	name := args[0]

	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	opts := workspace.CreateOptions{
		Name: name,

		BaseBranch: createBaseBranch,

		Branch: createBranch,

		AgentID: createAgentID,

		Description: createDescription,
	}

	ws, err := manager.Create(opts)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	ui.Success("Created workspace: %s", ws.Name)

	if ws.Index != "" {
		ui.Info("ID: %s", ws.Index)
	} else {
		ui.Info("ID: %s", ws.ID)
	}

	ui.Info("Branch: %s", ws.Branch)

	ui.Info("Path: %s", ws.Path)

	if ws.AgentID != "" {
		ui.Info("Assigned to agent: %s", ws.AgentID)
	}

	// Execute hooks unless --no-hooks was specified
	if !createNoHooks {
		if err := executeWorkspaceHooks(ws, hooks.EventWorkspaceCreate); err != nil {
			// Hooks failed but workspace was created
			ui.Error("Hook execution failed: %v", err)
			ui.Warning("Workspace was created but hooks did not run successfully")
			return nil
		}
	}

	return nil
}

func runListWorkspace(cmd *cobra.Command, args []string) error {
	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	workspaces, err := manager.List(workspace.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	// Handle JSON output
	if ui.GlobalFormatter.IsJSON() {
		return ui.GlobalFormatter.Output(workspaces)
	}

	// Handle pretty output
	if listOneline {
		// One line per workspace for fzf integration

		for _, ws := range workspaces {

			// Format: name<tab>id<tab>branch<tab>status<tab>path<tab>description

			description := ws.Description

			if description == "" {
				description = "-"
			}

			id := ws.ID
			if ws.Index != "" {
				id = ws.Index
			}
			ui.PrintTSV([][]string{{ws.Name, id, ws.Branch, ws.Status.String(), ws.Path, description}})

		}
	} else {
		ui.PrintWorkspaceList(workspaces)
	}

	return nil
}

func runShowWorkspace(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	// Resolve workspace by name or ID
	ws, err := manager.ResolveWorkspace(identifier)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Handle JSON output
	if ui.GlobalFormatter.IsJSON() {
		return ui.GlobalFormatter.Output(ws)
	}

	// Print detailed workspace information
	ui.PrintWorkspaceDetails(ws)

	return nil
}

func runRemoveWorkspace(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	// Resolve workspace by name or ID

	ws, err := manager.ResolveWorkspace(identifier)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Check if current working directory is inside the workspace
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Resolve current directory to handle OS-level symlinks (e.g., macOS /var -> /private/var)
	resolvedCwd, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		// If we can't resolve, use original path
		resolvedCwd = cwd
	}

	// Check both original and resolved paths
	if strings.HasPrefix(cwd, ws.Path) || strings.HasPrefix(resolvedCwd, ws.Path) {
		ui.Error("Cannot remove workspace while working inside it")
		ui.Info("Please change to a different directory first:")
		projectRoot, _ := config.FindProjectRoot()
		if projectRoot != "" {
			ui.Info("  cd %s", projectRoot)
		}
		return fmt.Errorf("cannot remove workspace from within itself")
	}

	if !removeForce {

		ui.Warning("This will remove workspace '%s' (%s) and its branch '%s'", ws.Name, ws.ID, ws.Branch)

		response := ui.Prompt("Are you sure? (y/N): ")

		if response != "y" && response != "Y" {
			ui.Info("Removal cancelled")
			return nil
		}

	}

	// Execute pre-removal hooks unless --no-hooks was specified
	if !removeNoHooks {
		if err := executeWorkspaceHooks(ws, hooks.EventWorkspaceRemove); err != nil {
			ui.Error("Hook execution failed: %v", err)
			ui.Warning("Stopping removal due to hook failure")
			return err
		}
	}

	if err := manager.Remove(ws.ID); err != nil {
		return fmt.Errorf("failed to remove workspace: %w", err)
	}

	ui.Success("Removed workspace: %s (%s)", ws.Name, ws.ID)

	return nil
}

func runPruneWorkspace(cmd *cobra.Command, args []string) error {
	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	opts := workspace.CleanupOptions{
		Days: pruneDays,

		DryRun: pruneDryRun,
	}

	removed, err := manager.Cleanup(opts)
	if err != nil {
		return fmt.Errorf("failed to prune workspaces: %w", err)
	}

	if len(removed) == 0 {

		ui.Info("No workspaces to prune")

		return nil

	}

	if pruneDryRun {
		ui.Info("Would remove %d workspace(s):", len(removed))
	} else {
		ui.Success("Removed %d workspace(s):", len(removed))
	}

	for _, id := range removed {
		ui.OutputLine("  - %s", id)
	}

	return nil
}

func runCdWorkspace(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	// Resolve workspace by name or ID
	ws, err := manager.ResolveWorkspace(identifier)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}

	// Get user's shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
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
	ui.Info("Entering workspace: %s", ws.Name)
	ui.Info("Path: %s", ws.Path)
	ui.Info("Exit the shell to return to your original directory")

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

func getWorkspaceManager() (*workspace.Manager, error) {
	// Find project root

	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return nil, err
	}

	// Create configuration manager

	configManager := config.NewManager(projectRoot)

	// Ensure initialized

	if !configManager.IsInitialized() {
		return nil, fmt.Errorf("amux not initialized. Run 'amux init' first")
	}

	// Create workspace manager

	return workspace.NewManager(configManager)
}

// executeWorkspaceHooks runs hooks for the given workspace event
func executeWorkspaceHooks(ws *workspace.Workspace, event hooks.Event) error {
	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return err
	}

	configManager := config.NewManager(projectRoot)
	configDir := configManager.GetAmuxDir()

	// Load hooks configuration
	hooksConfig, err := hooks.LoadConfig(configDir)
	if err != nil {
		return fmt.Errorf("failed to load hooks: %w", err)
	}

	// Get hooks for this event
	eventHooks := hooksConfig.GetHooksForEvent(event)
	if len(eventHooks) == 0 {
		return nil // No hooks configured
	}

	// Check if hooks are trusted
	trusted, err := hooks.IsTrusted(configDir, hooksConfig)
	if err != nil {
		return fmt.Errorf("failed to check hook trust: %w", err)
	}

	if !trusted {
		ui.Warning("This project has hooks configured but they are not trusted.")
		ui.Info("Run 'amux hooks trust' to trust hooks in this project.")
		return nil
	}

	// Prepare environment variables
	env := map[string]string{
		"AMUX_WORKSPACE_ID":          ws.ID,
		"AMUX_WORKSPACE_NAME":        ws.Name,
		"AMUX_WORKSPACE_PATH":        ws.Path,
		"AMUX_WORKSPACE_BRANCH":      ws.Branch,
		"AMUX_WORKSPACE_BASE_BRANCH": ws.BaseBranch,
		"AMUX_EVENT":                 string(event),
		"AMUX_EVENT_TIME":            time.Now().Format(time.RFC3339),
		"AMUX_PROJECT_ROOT":          projectRoot,
		"AMUX_CONFIG_DIR":            configDir,
	}

	if ws.AgentID != "" {
		env["AMUX_AGENT_ID"] = ws.AgentID
	}

	// Execute hooks
	executor := hooks.NewExecutor(configDir, env)
	return executor.ExecuteHooks(event, eventHooks)
}
