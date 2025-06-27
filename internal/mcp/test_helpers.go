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

	// Create default config with test agents and save it
	cfg := config.DefaultConfig()

	// Add test agent for testing
	cfg.Agents["test-agent"] = config.Agent{
		Name: "Test Agent",
		Type: config.AgentTypeTmux,
		Environment: map[string]string{
			"TEST_ENV": "test",
		},
		Params: &config.TmuxParams{
			Command: config.Command{Single: "echo 'test agent'"},
		},
	}

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
