package mcp

import (
	"testing"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/tests/helpers"
)

// setupTestServer creates a test MCP server with a temporary git repository
func setupTestServer(t *testing.T) *ServerV2 {
	t.Helper()

	// Create test repo
	testRepoPath := helpers.CreateTestRepo(t)

	// Create config manager
	configManager := config.NewManager(testRepoPath)

	// Create default config and save it
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// Create server
	server, err := NewServerV2(configManager, "stdio", nil)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Cleanup is handled by helpers.CreateTestRepo

	return server

}
