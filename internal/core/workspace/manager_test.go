package workspace_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/git"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/tests/helpers"
	"gopkg.in/yaml.v3"
)

func TestManager_CreateWithExistingBranch(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

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
		BranchMode:  workspace.BranchModeCheckout, // Explicitly use existing branch
		Description: "Test workspace with existing branch",
	}

	ws, err := manager.Create(context.Background(), opts)
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

	// Verify we can get the workspace back
	retrievedWs, err := manager.Get(context.Background(), workspace.ID(ws.ID))
	if err != nil {
		t.Errorf("Failed to retrieve workspace: %v", err)
	}
	if retrievedWs.ID != ws.ID {
		t.Errorf("Retrieved workspace ID mismatch: got %s, want %s", retrievedWs.ID, ws.ID)
	}

	// Clean up
	err = manager.Remove(context.Background(), workspace.Identifier(ws.ID), workspace.RemoveOptions{})
	if err != nil {
		t.Fatalf("Failed to remove workspace: %v", err)
	}
}

func TestManager_CreateWithNewBranch(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

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

	ws, err := manager.Create(context.Background(), opts)
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
	err = manager.Remove(context.Background(), workspace.Identifier(ws.ID), workspace.RemoveOptions{})
	if err != nil {
		t.Fatalf("Failed to remove workspace: %v", err)
	}
}

func TestManager_RemoveWithManuallyDeletedWorktree(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

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

	ws, err := manager.Create(context.Background(), opts)
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
	err = manager.Remove(context.Background(), workspace.Identifier(ws.ID), workspace.RemoveOptions{})
	if err != nil {
		t.Fatalf("Failed to remove workspace with manually deleted worktree: %v", err)
	}

	// Verify workspace is no longer listed
	workspaces, err := manager.List(context.Background(), workspace.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list workspaces: %v", err)
	}

	for _, w := range workspaces {
		if w.ID == ws.ID {
			t.Errorf("Workspace %s still exists in list after removal", ws.ID)
		}
	}
}

func TestManager_ConsistencyChecking(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

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

	// Test Case 1: Consistent workspace
	t.Run("ConsistentWorkspace", func(t *testing.T) {
		opts := workspace.CreateOptions{
			Name:        "test-consistent",
			BaseBranch:  "main",
			Description: "Test consistent workspace",
		}

		ws, err := manager.Create(context.Background(), opts)
		if err != nil {
			t.Fatalf("Failed to create workspace: %v", err)
		}
		t.Cleanup(func() {
			if err := manager.Remove(context.Background(), workspace.Identifier(ws.ID), workspace.RemoveOptions{}); err != nil {
				t.Logf("Failed to remove workspace %s: %v", ws.ID, err)
			}
		})

		// Get workspace and check consistency
		retrieved, err := manager.Get(context.Background(), workspace.ID(ws.ID))
		if err != nil {
			t.Fatalf("Failed to get workspace: %v", err)
		}

		t.Logf("Workspace path: %s", retrieved.Path)
		t.Logf("Path exists: %v, Worktree exists: %v, Status: %s",
			retrieved.PathExists, retrieved.WorktreeExists, retrieved.Status)

		if retrieved.Status != workspace.StatusConsistent {
			t.Errorf("Expected status StatusConsistent, got '%s'", retrieved.Status)
		}
		if !retrieved.PathExists {
			t.Error("Expected PathExists to be true")
		}
		if !retrieved.WorktreeExists {
			t.Error("Expected WorktreeExists to be true")
		}
	})

	// Test Case 2: Folder deleted but git worktree exists
	t.Run("FolderMissing", func(t *testing.T) {
		opts := workspace.CreateOptions{
			Name:        "test-folder-missing",
			BaseBranch:  "main",
			Description: "Test folder missing workspace",
		}

		ws, err := manager.Create(context.Background(), opts)
		if err != nil {
			t.Fatalf("Failed to create workspace: %v", err)
		}
		t.Cleanup(func() {
			if err := manager.Remove(context.Background(), workspace.Identifier(ws.ID), workspace.RemoveOptions{}); err != nil {
				t.Logf("Failed to remove workspace %s: %v", ws.ID, err)
			}
		})

		// Manually delete the folder
		err = os.RemoveAll(ws.Path)
		if err != nil {
			t.Fatalf("Failed to delete folder: %v", err)
		}

		// Get workspace and check consistency
		retrieved, err := manager.Get(context.Background(), workspace.ID(ws.ID))
		if err != nil {
			t.Fatalf("Failed to get workspace: %v", err)
		}

		// List worktrees to debug
		gitOps := git.NewOperations(repoDir)
		worktrees, _ := gitOps.ListWorktrees()
		t.Logf("Worktrees after folder deletion:")
		for _, wt := range worktrees {
			t.Logf("  - Path: %s, Branch: %s", wt.Path, wt.Branch)
		}

		t.Logf("Workspace path: %s", retrieved.Path)
		t.Logf("Path exists: %v, Worktree exists: %v, Status: %s",
			retrieved.PathExists, retrieved.WorktreeExists, retrieved.Status)

		if retrieved.Status != workspace.StatusFolderMissing {
			t.Errorf("Expected status StatusFolderMissing, got '%s'", retrieved.Status)
		}
		if retrieved.PathExists {
			t.Error("Expected PathExists to be false")
		}
		if !retrieved.WorktreeExists {
			t.Error("Expected WorktreeExists to be true")
		}
	})

	// Test Case 3: Git worktree removed but folder exists
	// This simulates user running `git worktree remove` directly,
	// but amux metadata still exists
	t.Run("WorktreeMissing", func(t *testing.T) {
		// First create a normal workspace
		opts := workspace.CreateOptions{
			Name:        "test-worktree-missing-temp",
			BaseBranch:  "main",
			Description: "Temporary workspace",
		}

		tempWs, err := manager.Create(context.Background(), opts)
		if err != nil {
			t.Fatalf("Failed to create temp workspace: %v", err)
		}

		// Get the workspace metadata to simulate orphaned metadata
		tempPath := filepath.Join(configManager.GetWorkspacesDir(), tempWs.ID, "workspace.yaml")
		metadata, err := os.ReadFile(tempPath)
		if err != nil {
			t.Fatalf("Failed to read workspace metadata: %v", err)
		}

		// Remove the temp workspace properly
		manager.Remove(context.Background(), workspace.Identifier(tempWs.ID), workspace.RemoveOptions{})

		// Now create the test scenario:
		// 1. Create workspace directory structure manually
		wsID := "workspace-test-worktree-missing-manual"
		wsDir := filepath.Join(configManager.GetWorkspacesDir(), wsID)
		wsPath := filepath.Join(wsDir, "worktree")
		err = os.MkdirAll(wsPath, 0o755)
		if err != nil {
			t.Fatalf("Failed to create workspace folder: %v", err)
		}

		// 2. Create metadata file pointing to this folder
		ws := &workspace.Workspace{
			ID:          wsID,
			Name:        "test-worktree-missing",
			Branch:      "amux/" + wsID,
			BaseBranch:  "main",
			Path:        wsPath,
			Description: "Test worktree missing workspace",
			CreatedAt:   tempWs.CreatedAt,
		}

		// Save the metadata in the new location
		metadataPath := filepath.Join(wsDir, "workspace.yaml")
		modifiedMetadata := string(metadata)
		// Update metadata with new workspace info
		// This is a simple approach - in reality we'd marshal the struct
		err = os.WriteFile(metadataPath, []byte(modifiedMetadata), 0o644)
		if err != nil {
			t.Fatalf("Failed to write workspace metadata: %v", err)
		}

		// Update the metadata with correct values
		data, _ := yaml.Marshal(ws)
		os.WriteFile(metadataPath, data, 0o644)

		// Get workspace and check consistency
		retrieved, err := manager.Get(context.Background(), workspace.ID(wsID))
		if err != nil {
			t.Fatalf("Failed to get workspace: %v", err)
		}

		t.Logf("Workspace path: %s", retrieved.Path)
		t.Logf("Path exists: %v, Worktree exists: %v, Status: %s",
			retrieved.PathExists, retrieved.WorktreeExists, retrieved.Status)

		if retrieved.Status != workspace.StatusWorktreeMissing {
			t.Errorf("Expected status StatusWorktreeMissing, got '%s'", retrieved.Status)
		}
		if !retrieved.PathExists {
			t.Error("Expected PathExists to be true")
		}
		if retrieved.WorktreeExists {
			t.Error("Expected WorktreeExists to be false")
		}

		// Clean up
		os.RemoveAll(wsPath)
		os.Remove(metadataPath)
	})
}

func TestManager_CreateSetsContextPath(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

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

	// Create workspace
	opts := workspace.CreateOptions{
		Name:        "test-context-path",
		BaseBranch:  "main",
		Description: "Test workspace context path",
	}

	ws, err := manager.Create(context.Background(), opts)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Verify storage path is set
	if ws.StoragePath == "" {
		t.Error("Expected StoragePath to be set, but it was empty")
	}

	// Verify storage path follows expected pattern
	expectedStoragePath := filepath.Join(configManager.GetWorkspacesDir(), ws.ID, "storage")
	if ws.StoragePath != expectedStoragePath {
		t.Errorf("Expected storage path %s, got %s", expectedStoragePath, ws.StoragePath)
	}

	// Verify storage directory was created
	if _, err := os.Stat(ws.StoragePath); os.IsNotExist(err) {
		t.Error("Expected storage directory to be created, but it doesn't exist")
	}

	// Verify we can retrieve the workspace with context path
	retrievedWs, err := manager.Get(context.Background(), workspace.ID(ws.ID))
	if err != nil {
		t.Fatalf("Failed to retrieve workspace: %v", err)
	}

	if retrievedWs.StoragePath != ws.StoragePath {
		t.Errorf("Retrieved workspace storage path mismatch: got %s, want %s",
			retrievedWs.StoragePath, ws.StoragePath)
	}

	// Clean up
	err = manager.Remove(context.Background(), workspace.Identifier(ws.ID), workspace.RemoveOptions{})
	if err != nil {
		t.Fatalf("Failed to remove workspace: %v", err)
	}
}

func TestManager_CreateWithNewBranchFlag(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

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

	t.Run("CreateNewBranch", func(t *testing.T) {
		// Create workspace with new branch using -b flag
		opts := workspace.CreateOptions{
			Name:        "test-new-branch",
			Branch:      "feature/test-new",
			BranchMode:  workspace.BranchModeCreate,
			Description: "Test workspace with new branch flag",
		}

		ws, err := manager.Create(context.Background(), opts)
		if err != nil {
			t.Fatalf("Failed to create workspace: %v", err)
		}

		// Verify workspace was created with specified branch
		if ws.Branch != "feature/test-new" {
			t.Errorf("Expected branch feature/test-new, got %s", ws.Branch)
		}

		// Clean up
		err = manager.Remove(context.Background(), workspace.Identifier(ws.ID), workspace.RemoveOptions{})
		if err != nil {
			t.Fatalf("Failed to remove workspace: %v", err)
		}
	})

	t.Run("CreateNewBranchAlreadyExists", func(t *testing.T) {
		// Create a branch first
		gitOps := git.NewOperations(repoDir)
		existingBranch := "feature/already-exists"
		err := gitOps.CreateBranch(existingBranch, "main")
		if err != nil {
			t.Fatalf("Failed to create existing branch: %v", err)
		}

		// Try to create workspace with -b flag for existing branch
		opts := workspace.CreateOptions{
			Name:       "test-new-branch-fail",
			Branch:     existingBranch,
			BranchMode: workspace.BranchModeCreate,
		}

		_, err = manager.Create(context.Background(), opts)
		if err == nil {
			t.Fatal("Expected error when creating new branch that already exists")
		}

		expectedError := "cannot create branch 'feature/already-exists': already exists. Use -c to checkout existing branch"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})
}

func TestManager_CreateWithCheckoutFlag(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

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

	t.Run("CheckoutExistingBranch", func(t *testing.T) {
		// Create an existing branch first
		gitOps := git.NewOperations(repoDir)
		existingBranch := "feature/to-checkout"
		err := gitOps.CreateBranch(existingBranch, "main")
		if err != nil {
			t.Fatalf("Failed to create existing branch: %v", err)
		}

		// Create workspace with -c flag
		opts := workspace.CreateOptions{
			Name:        "test-checkout",
			Branch:      existingBranch,
			BranchMode:  workspace.BranchModeCheckout,
			Description: "Test workspace with checkout flag",
		}

		ws, err := manager.Create(context.Background(), opts)
		if err != nil {
			t.Fatalf("Failed to create workspace: %v", err)
		}

		// Verify workspace was created with existing branch
		if ws.Branch != existingBranch {
			t.Errorf("Expected branch %s, got %s", existingBranch, ws.Branch)
		}

		// Clean up
		err = manager.Remove(context.Background(), workspace.Identifier(ws.ID), workspace.RemoveOptions{})
		if err != nil {
			t.Fatalf("Failed to remove workspace: %v", err)
		}
	})

	t.Run("CheckoutNonExistentBranch", func(t *testing.T) {
		// Try to checkout non-existent branch
		opts := workspace.CreateOptions{
			Name:       "test-checkout-fail",
			Branch:     "feature/does-not-exist",
			BranchMode: workspace.BranchModeCheckout,
		}

		_, err = manager.Create(context.Background(), opts)
		if err == nil {
			t.Fatal("Expected error when checking out non-existent branch")
		}

		expectedError := "cannot checkout 'feature/does-not-exist': branch does not exist. Use -b to create new branch"
		if err.Error() != expectedError {
			t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})
}
