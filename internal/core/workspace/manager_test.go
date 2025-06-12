package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/git"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/tests/helpers"
)

func TestManager_CreateWithExistingBranch(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)
	defer os.RemoveAll(repoDir)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	// Create default config and save it
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Create workspace manager
	manager, err := workspace.NewManager(configManager)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create an existing branch first
	gitOps := git.NewOperations(repoDir)
	existingBranch := "feature/existing-branch"
	err = gitOps.CreateBranch(existingBranch, "main")
	if err != nil {
		t.Fatalf("Failed to create existing branch: %v", err)
	}

	// Create workspace using existing branch
	opts := workspace.CreateOptions{
		Name:        "test-existing",
		Branch:      existingBranch,
		Description: "Test workspace with existing branch",
	}

	ws, err := manager.Create(opts)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Verify workspace was created with the existing branch
	if ws.Branch != existingBranch {
		t.Errorf("Expected branch %s, got %s", existingBranch, ws.Branch)
	}

	// Verify worktree exists
	if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
		t.Errorf("Workspace path does not exist: %s", ws.Path)
	}

	// Verify .amux directory was created
	amuxDir := filepath.Join(ws.Path, ".amux")
	if _, err := os.Stat(amuxDir); os.IsNotExist(err) {
		t.Errorf("Amux directory not created in workspace")
	}

	// Clean up
	err = manager.Remove(ws.ID)
	if err != nil {
		t.Fatalf("Failed to remove workspace: %v", err)
	}
}

func TestManager_CreateWithNewBranch(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)
	defer os.RemoveAll(repoDir)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	// Create default config and save it
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Create workspace manager
	manager, err := workspace.NewManager(configManager)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create workspace without specifying branch (should create new branch)
	opts := workspace.CreateOptions{
		Name:        "test-new",
		BaseBranch:  "main",
		Description: "Test workspace with new branch",
	}

	ws, err := manager.Create(opts)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Verify workspace was created with new branch format
	expectedPrefix := "amux/workspace-"
	if ws.Branch[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected branch to start with %s, got %s", expectedPrefix, ws.Branch)
	}

	// Verify worktree exists
	if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
		t.Errorf("Workspace path does not exist: %s", ws.Path)
	}

	// Clean up
	err = manager.Remove(ws.ID)
	if err != nil {
		t.Fatalf("Failed to remove workspace: %v", err)
	}
}

func TestManager_RemoveWithManuallyDeletedWorktree(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)
	defer os.RemoveAll(repoDir)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Create workspace manager
	manager, err := workspace.NewManager(configManager)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create a workspace
	opts := workspace.CreateOptions{
		Name:        "test-manual-delete",
		BaseBranch:  "main",
		Description: "Test workspace for manual deletion",
	}

	ws, err := manager.Create(opts)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Verify workspace exists
	if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
		t.Errorf("Workspace path does not exist: %s", ws.Path)
	}

	// Manually delete the worktree directory (simulating user action)
	err = os.RemoveAll(ws.Path)
	if err != nil {
		t.Fatalf("Failed to manually delete worktree: %v", err)
	}

	// Verify worktree is gone
	if _, err := os.Stat(ws.Path); !os.IsNotExist(err) {
		t.Errorf("Worktree directory still exists after manual deletion")
	}

	// Now try to remove the workspace through amux
	// This should succeed and clean up metadata even though worktree is gone
	err = manager.Remove(ws.ID)
	if err != nil {
		t.Fatalf("Failed to remove workspace with manually deleted worktree: %v", err)
	}

	// Verify workspace is no longer listed
	workspaces, err := manager.List(workspace.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list workspaces: %v", err)
	}

	for _, w := range workspaces {
		if w.ID == ws.ID {
			t.Errorf("Workspace %s still exists in list after removal", ws.ID)
		}
	}
}
