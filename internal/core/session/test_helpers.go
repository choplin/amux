package session

import (
	"fmt"
	"testing"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/workspace"
	runtimeinit "github.com/aki/amux/internal/runtime/init"
	"github.com/aki/amux/internal/tests/helpers"
)

// setupTestEnvironment creates a test environment with workspace manager
func setupTestEnvironment(t *testing.T) (string, *workspace.Manager, *config.Manager) {
	// Initialize runtime registry for tests
	if err := runtimeinit.RegisterDefaults(); err != nil {
		t.Fatalf("Failed to initialize runtimes: %v", err)
	}

	// Create test repository
	repoPath := helpers.CreateTestRepo(t)

	// Create config manager
	configManager := config.NewManager(repoPath)

	// Create default config with test agents
	cfg := config.DefaultConfig()
	cfg.Agents["test-agent"] = config.Agent{
		Name:    "Test Agent",
		Runtime: "tmux",
		Command: []string{"echo", "test"},
	}
	// Add numbered agents for tests that use them
	for i := 0; i < 5; i++ {
		cfg.Agents[fmt.Sprintf("agent-%d", i)] = config.Agent{
			Name:    fmt.Sprintf("Agent %d", i),
			Runtime: "tmux",
			Command: []string{"echo", "test"},
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
