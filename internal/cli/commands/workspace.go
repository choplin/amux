package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aki/agentcave/internal/cli/ui"
	"github.com/aki/agentcave/internal/core/config"
	"github.com/aki/agentcave/internal/core/workspace"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage AgentCave workspaces",
	Long:  "Create, list, activate, deactivate, and remove isolated development workspaces",
}

var (
	// Create flags
	createBaseBranch  string
	createBranch      string
	createAgentID     string
	createDescription string

	// List flags
	listOneline bool

	// Cleanup flags
	cleanupDays   int
	cleanupDryRun bool

	// Remove flags
	removeForce bool
)

func init() {
	// Add subcommands
	workspaceCmd.AddCommand(createWorkspaceCmd)
	workspaceCmd.AddCommand(listWorkspaceCmd)
	workspaceCmd.AddCommand(removeWorkspaceCmd)
	workspaceCmd.AddCommand(cleanupWorkspaceCmd)

	// Create command flags
	createWorkspaceCmd.Flags().StringVarP(&createBaseBranch, "base-branch", "b", "", "Base branch to create workspace from")
	createWorkspaceCmd.Flags().StringVar(&createBranch, "branch", "", "Use existing branch instead of creating new one")
	createWorkspaceCmd.Flags().StringVarP(&createAgentID, "agent", "a", "", "Agent ID to assign to workspace")
	createWorkspaceCmd.Flags().StringVarP(&createDescription, "description", "d", "", "Description of the workspace")

	// List command flags
	listWorkspaceCmd.Flags().BoolVar(&listOneline, "oneline", false, "Show one workspace per line (for use with fzf)")

	// Cleanup command flags
	cleanupWorkspaceCmd.Flags().IntVarP(&cleanupDays, "days", "d", 7, "Remove workspaces idle for more than N days")
	cleanupWorkspaceCmd.Flags().BoolVar(&cleanupDryRun, "dry-run", false, "Show what would be removed without removing")

	// Remove command flags
	removeWorkspaceCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Force removal without confirmation")
}

var createWorkspaceCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new workspace",
	Args:  cobra.ExactArgs(1),
	RunE:  runCreateWorkspace,
}

var listWorkspaceCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces",
	Long: `List all workspaces in the project.

Examples:
  # List workspaces with detailed view
  agentcave workspace list

  # List workspaces in oneline format for scripting
  agentcave workspace list --oneline

  # Use with fzf to select a workspace
  agentcave workspace list --oneline | fzf | cut -f1

  # Remove selected workspace with fzf
  agentcave workspace remove $(agentcave workspace list --oneline | fzf | cut -f1)`,
	RunE: runListWorkspace,
}

var removeWorkspaceCmd = &cobra.Command{
	Use:   "remove <workspace-name-or-id>",
	Short: "Remove a workspace by name or ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemoveWorkspace,
}

var cleanupWorkspaceCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove old idle workspaces",
	RunE:  runCleanupWorkspace,
}

func runCreateWorkspace(cmd *cobra.Command, args []string) error {
	name := args[0]

	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	opts := workspace.CreateOptions{
		Name:        name,
		BaseBranch:  createBaseBranch,
		Branch:      createBranch,
		AgentID:     createAgentID,
		Description: createDescription,
	}

	ws, err := manager.Create(opts)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	ui.Success("Created workspace: %s", ws.Name)
	ui.Info("ID: %s", ws.ID)
	ui.Info("Branch: %s", ws.Branch)
	ui.Info("Path: %s", ws.Path)
	if ws.AgentID != "" {
		ui.Info("Assigned to agent: %s", ws.AgentID)
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

	if listOneline {
		// One line per workspace for fzf integration
		for _, ws := range workspaces {
			// Format: name<tab>id<tab>branch<tab>path<tab>description
			description := ws.Description
			if description == "" {
				description = "-"
			}
			fmt.Printf("%s\t%s\t%s\t%s\t%s\n", ws.Name, ws.ID, ws.Branch, ws.Path, description)
		}
	} else {
		ui.PrintWorkspaceList(workspaces)
	}
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

	if !removeForce {
		ui.Warning("This will remove workspace '%s' (%s) and its branch '%s'", ws.Name, ws.ID, ws.Branch)
		fmt.Print("Are you sure? (y/N): ")

		var response string
		_, _ = fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			ui.Info("Removal cancelled")
			return nil
		}
	}

	if err := manager.Remove(ws.ID); err != nil {
		return fmt.Errorf("failed to remove workspace: %w", err)
	}

	ui.Success("Removed workspace: %s (%s)", ws.Name, ws.ID)
	return nil
}

func runCleanupWorkspace(cmd *cobra.Command, args []string) error {
	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	opts := workspace.CleanupOptions{
		Days:   cleanupDays,
		DryRun: cleanupDryRun,
	}

	removed, err := manager.Cleanup(opts)
	if err != nil {
		return fmt.Errorf("failed to cleanup workspaces: %w", err)
	}

	if len(removed) == 0 {
		ui.Info("No workspaces to cleanup")
		return nil
	}

	if cleanupDryRun {
		ui.Info("Would remove %d workspace(s):", len(removed))
	} else {
		ui.Success("Removed %d workspace(s):", len(removed))
	}

	for _, id := range removed {
		fmt.Printf("  - %s\n", id)
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
		return nil, fmt.Errorf("AgentCave not initialized. Run 'agentcave init' first")
	}

	// Create workspace manager
	return workspace.NewManager(configManager)
}
