package session

import (
	"fmt"
	"testing"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/tests/helpers"
)

// setupTestEnvironment creates a test environment with workspace manager
func setupTestEnvironment(t *testing.T) (string, *workspace.Manager, *config.Manager) {
	// Create test repository
	repoPath := helpers.CreateTestRepo(t)

	// Create config manager
	configManager := config.NewManager(repoPath)

	// Create default config with test agents
	cfg := config.DefaultConfig()
	cfg.Agents["test-agent"] = config.Agent{
		Name: "Test Agent",
		Type: config.AgentTypeTmux,
		Params: &config.TmuxParams{
			Command: config.Command{Single: "echo test"},
		},
	}
	// Add numbered agents for tests that use them
	for i := 0; i < 5; i++ {
		cfg.Agents[fmt.Sprintf("agent-%d", i)] = config.Agent{
			Name: fmt.Sprintf("Agent %d", i),
			Type: config.AgentTypeTmux,
			Params: &config.TmuxParams{
				Command: config.Command{Single: "echo test"},
			},
		}
	}
	if err := configManager.Save(cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	if err != nil {
		t.Fatalf("Failed to create workspace manager: %v", err)
	}

	return repoPath, wsManager, configManager
}
