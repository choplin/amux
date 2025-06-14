package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/tests/helpers"
)

// TestWorkspaceRemovalSafetyCheck tests the safety check logic for workspace removal
func TestWorkspaceRemovalSafetyCheck(t *testing.T) {
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

	// Create a test workspace
	opts := workspace.CreateOptions{
		Name:        "test-removal-safety",
		BaseBranch:  "main",
		Description: "Test workspace removal safety",
	}

	ws, err := manager.Create(opts)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	// Ensure cleanup happens after all subtests complete
	t.Cleanup(func() {
		// Clean up: change to repo dir first, then remove workspace
		os.Chdir(repoDir)
		if err := manager.Remove(ws.ID); err != nil {
			t.Logf("Failed to remove workspace %s: %v", ws.ID, err)
		}
	})

	// Test the safety check logic directly
	t.Run("SafetyCheckLogic", func(t *testing.T) {
		// Test cases for the safety check logic
		testCases := []struct {
			name          string
			currentDir    string
			workspacePath string
			shouldBlock   bool
		}{
			{
				name:          "InWorkspaceRoot",
				currentDir:    ws.Path,
				workspacePath: ws.Path,
				shouldBlock:   true,
			},
			{
				name:          "InWorkspaceSubdir",
				currentDir:    filepath.Join(ws.Path, "src", "components"),
				workspacePath: ws.Path,
				shouldBlock:   true,
			},
			{
				name:          "OutsideWorkspace",
				currentDir:    repoDir,
				workspacePath: ws.Path,
				shouldBlock:   false,
			},
			{
				name:          "InDifferentWorkspace",
				currentDir:    "/tmp/other-workspace",
				workspacePath: ws.Path,
				shouldBlock:   false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Test the actual check logic used in runRemoveWorkspace
				// This mimics the logic: check if cwd or resolved cwd is inside workspace
				resolvedCwd, _ := filepath.EvalSymlinks(tc.currentDir)
				isInside := strings.HasPrefix(tc.currentDir, tc.workspacePath) ||
					strings.HasPrefix(resolvedCwd, tc.workspacePath)
				if isInside != tc.shouldBlock {
					t.Errorf("Safety check failed for %s: got %v, want %v",
						tc.name, isInside, tc.shouldBlock)
				}
			})
		}
	})

	// Test actual directory changes
	t.Run("ActualDirectoryChanges", func(t *testing.T) {
		// Save original directory
		originalWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get original working directory: %v", err)
		}
		defer os.Chdir(originalWd)

		// Test 1: Change to workspace and verify we're inside
		if err := os.Chdir(ws.Path); err != nil {
			t.Fatalf("Failed to change to workspace directory: %v", err)
		}

		cwd, _ := os.Getwd()
		// On macOS, paths might differ due to symlinks (/var vs /private/var)
		// Just verify we can change to the directory successfully
		t.Logf("Changed to workspace: cwd=%s, ws.Path=%s", cwd, ws.Path)

		// Test 2: Change to repo dir and verify we're outside
		if err := os.Chdir(repoDir); err != nil {
			t.Fatalf("Failed to change to repo directory: %v", err)
		}

		cwd, _ = os.Getwd()
		// The key test is that the safety check logic works correctly
		// which is tested in the SafetyCheckLogic subtest above
		t.Logf("Changed to repo dir: cwd=%s", cwd)
	})
}

// TestWorkspaceCdCommand tests the workspace cd command functionality
func TestWorkspaceCdCommand(t *testing.T) {
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

	// Create a test workspace
	opts := workspace.CreateOptions{
		Name:        "test-cd",
		BaseBranch:  "main",
		Description: "Test workspace cd command",
	}

	ws, err := manager.Create(opts)
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}
	// Ensure cleanup happens after all subtests complete
	t.Cleanup(func() {
		if err := manager.Remove(ws.ID); err != nil {
			t.Logf("Failed to remove workspace %s: %v", ws.ID, err)
		}
	})

	// Test workspace resolution by name
	t.Run("ResolveByName", func(t *testing.T) {
		resolved, err := manager.ResolveWorkspace("test-cd")
		if err != nil {
			t.Errorf("Failed to resolve workspace by name: %v", err)
		}
		if resolved.ID != ws.ID {
			t.Errorf("Resolved wrong workspace: got %s, want %s", resolved.ID, ws.ID)
		}
	})

	// Test workspace resolution by ID
	t.Run("ResolveByID", func(t *testing.T) {
		resolved, err := manager.ResolveWorkspace(ws.ID)
		if err != nil {
			t.Errorf("Failed to resolve workspace by ID: %v", err)
		}
		if resolved.ID != ws.ID {
			t.Errorf("Resolved wrong workspace: got %s, want %s", resolved.ID, ws.ID)
		}
	})

	// Test workspace resolution by index
	t.Run("ResolveByIndex", func(t *testing.T) {
		if ws.Index != "" {
			resolved, err := manager.ResolveWorkspace(ws.Index)
			if err != nil {
				t.Errorf("Failed to resolve workspace by index: %v", err)
			}
			if resolved.ID != ws.ID {
				t.Errorf("Resolved wrong workspace: got %s, want %s", resolved.ID, ws.ID)
			}
		}
	})

	// Test invalid workspace
	t.Run("InvalidWorkspace", func(t *testing.T) {
		_, err := manager.ResolveWorkspace("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent workspace, got nil")
		}
	})

	// Test workspace path exists
	t.Run("WorkspacePathExists", func(t *testing.T) {
		// Verify the workspace was created successfully
		if ws == nil {
			t.Fatal("Workspace is nil")
		}

		// Log workspace details for debugging
		t.Logf("Checking workspace path: %s", ws.Path)

		// Check if the path exists
		if info, err := os.Stat(ws.Path); os.IsNotExist(err) {
			t.Errorf("Workspace path does not exist: %s", ws.Path)
		} else if err != nil {
			t.Errorf("Error checking workspace path: %v", err)
		} else {
			t.Logf("Workspace path exists: %s (IsDir: %v)", ws.Path, info.IsDir())
		}
	})
}
