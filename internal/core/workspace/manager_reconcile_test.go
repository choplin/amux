package workspace_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/tests/helpers"
)

func TestManager_ListReconciliation(t *testing.T) {
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

	// Create ID mapper to check indices
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create some workspaces
	ctx := context.Background()
	ws1, err := manager.Create(ctx, workspace.CreateOptions{
		Name:        "test-ws1",
		Description: "Test workspace 1",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace 1: %v", err)
	}

	ws2, err := manager.Create(ctx, workspace.CreateOptions{
		Name:        "test-ws2",
		Description: "Test workspace 2",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace 2: %v", err)
	}

	ws3, err := manager.Create(ctx, workspace.CreateOptions{
		Name:        "test-ws3",
		Description: "Test workspace 3",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace 3: %v", err)
	}

	// Verify all have indices
	if _, exists := idMapper.GetWorkspaceIndex(ws1.ID); !exists {
		t.Error("Expected ws1 to have an index")
	}
	if _, exists := idMapper.GetWorkspaceIndex(ws2.ID); !exists {
		t.Error("Expected ws2 to have an index")
	}
	if _, exists := idMapper.GetWorkspaceIndex(ws3.ID); !exists {
		t.Error("Expected ws3 to have an index")
	}

	// Manually delete workspace 2's directory (simulating external deletion)
	workspaceDir := filepath.Join(configManager.GetWorkspacesDir(), ws2.ID)
	err = os.RemoveAll(workspaceDir)
	if err != nil {
		t.Fatalf("Failed to manually delete workspace 2: %v", err)
	}

	// List workspaces - this should trigger reconciliation
	workspaces, err := manager.List(ctx, workspace.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list workspaces: %v", err)
	}

	// Should only have 2 workspaces now
	if len(workspaces) != 2 {
		t.Errorf("Expected 2 workspaces after deletion, got %d", len(workspaces))
	}

	// Verify ws2's index was cleaned up
	if _, exists := idMapper.GetWorkspaceIndex(ws2.ID); exists {
		t.Error("Expected ws2 index to be removed after reconciliation")
	}

	// Verify ws1 and ws3 still have indices
	if _, exists := idMapper.GetWorkspaceIndex(ws1.ID); !exists {
		t.Error("Expected ws1 to still have an index")
	}
	if _, exists := idMapper.GetWorkspaceIndex(ws3.ID); !exists {
		t.Error("Expected ws3 to still have an index")
	}

	// Clean up
	manager.Remove(ctx, workspace.Identifier(ws1.ID))
	manager.Remove(ctx, workspace.Identifier(ws3.ID))
}

func TestManager_GetReconciliation(t *testing.T) {
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

	// Create ID mapper to check indices
	idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
	if err != nil {
		t.Fatalf("Failed to create ID mapper: %v", err)
	}

	// Create a workspace
	ctx := context.Background()
	ws, err := manager.Create(ctx, workspace.CreateOptions{
		Name:        "test-reconcile-get",
		Description: "Test get reconciliation",
	})
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	// Verify it has an index
	if _, exists := idMapper.GetWorkspaceIndex(ws.ID); !exists {
		t.Error("Expected workspace to have an index")
	}

	// Manually delete the workspace directory
	workspaceDir := filepath.Join(configManager.GetWorkspacesDir(), ws.ID)
	err = os.RemoveAll(workspaceDir)
	if err != nil {
		t.Fatalf("Failed to manually delete workspace: %v", err)
	}

	// Try to get the workspace - should fail but NOT clean up index (reconciliation only happens during List)
	_, err = manager.Get(ctx, workspace.ID(ws.ID))
	if err == nil {
		t.Error("Expected error when getting deleted workspace")
	}

	// Verify the index still exists (Get does not reconcile)
	if _, exists := idMapper.GetWorkspaceIndex(ws.ID); !exists {
		t.Error("Expected workspace index to still exist after failed Get (reconciliation only happens during List)")
	}

	// Now list workspaces - this should trigger reconciliation
	_, err = manager.List(ctx, workspace.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list workspaces: %v", err)
	}

	// Now the index should be cleaned up
	if _, exists := idMapper.GetWorkspaceIndex(ws.ID); exists {
		t.Error("Expected workspace index to be removed after List reconciliation")
	}
}
