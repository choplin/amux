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
	createAgentID     string
	createDescription string

	// List flags
	listStatus string

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
	workspaceCmd.AddCommand(activateWorkspaceCmd)
	workspaceCmd.AddCommand(deactivateWorkspaceCmd)
	workspaceCmd.AddCommand(removeWorkspaceCmd)
	workspaceCmd.AddCommand(cleanupWorkspaceCmd)

	// Create command flags
	createWorkspaceCmd.Flags().StringVarP(&createBaseBranch, "base-branch", "b", "", "Base branch to create workspace from")
	createWorkspaceCmd.Flags().StringVarP(&createAgentID, "agent", "a", "", "Agent ID to assign to workspace")
	createWorkspaceCmd.Flags().StringVarP(&createDescription, "description", "d", "", "Description of the workspace")

	// List command flags
	listWorkspaceCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status (active/idle)")

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
	RunE:  runListWorkspace,
}

var activateWorkspaceCmd = &cobra.Command{
	Use:   "activate <workspace-id>",
	Short: "Mark a workspace as active",
	Args:  cobra.ExactArgs(1),
	RunE:  runActivateWorkspace,
}

var deactivateWorkspaceCmd = &cobra.Command{
	Use:   "deactivate <workspace-id>",
	Short: "Mark a workspace as idle",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeactivateWorkspace,
}

var removeWorkspaceCmd = &cobra.Command{
	Use:   "remove <workspace-id>",
	Short: "Remove a workspace",
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
		AgentID:     createAgentID,
		Description: createDescription,
	}

	ws, err := manager.Create(opts)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	ui.Success("Created workspace: %s", ws.ID)
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

	opts := workspace.ListOptions{}
	if listStatus != "" {
		opts.Status = workspace.Status(listStatus)
	}

	workspaces, err := manager.List(opts)
	if err != nil {
		return fmt.Errorf("failed to list workspaces: %w", err)
	}

	ui.PrintWorkspaceList(workspaces)
	return nil
}

func runActivateWorkspace(cmd *cobra.Command, args []string) error {
	workspaceID := args[0]

	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	if err := manager.Activate(workspaceID); err != nil {
		return fmt.Errorf("failed to activate workspace: %w", err)
	}

	ui.Success("Activated workspace: %s", workspaceID)
	return nil
}

func runDeactivateWorkspace(cmd *cobra.Command, args []string) error {
	workspaceID := args[0]

	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	if err := manager.Deactivate(workspaceID); err != nil {
		return fmt.Errorf("failed to deactivate workspace: %w", err)
	}

	ui.Success("Deactivated workspace: %s", workspaceID)
	return nil
}

func runRemoveWorkspace(cmd *cobra.Command, args []string) error {
	workspaceID := args[0]

	manager, err := getWorkspaceManager()
	if err != nil {
		return err
	}

	// Get workspace details for confirmation
	ws, err := manager.Get(workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	if !removeForce {
		ui.Warning("This will remove workspace '%s' (%s) and its branch '%s'", ws.Name, ws.ID, ws.Branch)
		fmt.Print("Are you sure? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			ui.Info("Removal cancelled")
			return nil
		}
	}

	if err := manager.Remove(workspaceID); err != nil {
		return fmt.Errorf("failed to remove workspace: %w", err)
	}

	ui.Success("Removed workspace: %s", workspaceID)
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