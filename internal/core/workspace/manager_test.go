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
