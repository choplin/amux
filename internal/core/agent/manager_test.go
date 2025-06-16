package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aki/amux/internal/core/config"
)

func setupTestManager(t *testing.T) (*Manager, string) {
	tmpDir := t.TempDir()
	amuxDir := filepath.Join(tmpDir, ".amux")
	if err := os.MkdirAll(amuxDir, 0o755); err != nil {
		t.Fatalf("Failed to create amux dir: %v", err)
	}

	// Create config file with test agents
	testConfig := &config.Config{
		Version: "1.0",
		Project: config.ProjectConfig{
			Name:         "test-project",
			DefaultAgent: "claude",
		},
		MCP: config.MCPConfig{
			Transport: config.TransportConfig{
				Type: "stdio",
			},
		},
		Agents: map[string]config.Agent{
			"claude": {
				Name: "Claude",
				Type: config.AgentTypeTmux,
				Environment: map[string]string{
					"ANTHROPIC_API_KEY": "test-key",
				},
				Params: &config.TmuxParams{
					Command: "claude",
				},
			},
			"gpt": {
				Name: "GPT",
				Type: config.AgentTypeTmux,
				Params: &config.TmuxParams{
					Command: "gpt",
				},
			},
		},
	}

	configManager := config.NewManager(tmpDir)
	if err := configManager.Save(testConfig); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	return NewManager(configManager), tmpDir
}

func TestManager_GetAgent(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Test existing agent
	agent, err := manager.GetAgent("claude")
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	if agent.Name != "Claude" {
		t.Errorf("Expected agent name 'Claude', got '%s'", agent.Name)
	}

	// Check tmux options
	params, err := agent.GetTmuxParams()
	if err != nil {
		t.Errorf("Failed to get tmux options: %v", err)
	}
	if params.Command != "claude" {
		t.Errorf("Expected tmux command 'claude', got '%s'", params.Command)
	}

	if agent.Environment["ANTHROPIC_API_KEY"] != "test-key" {
		t.Errorf("Expected environment variable not found")
	}

	// Test non-existent agent
	_, err = manager.GetAgent("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent agent")
	}
}

func TestManager_ListAgents(t *testing.T) {
	manager, _ := setupTestManager(t)

	agents, err := manager.ListAgents()
	if err != nil {
		t.Fatalf("Failed to list agents: %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}

	if _, exists := agents["claude"]; !exists {
		t.Error("Expected 'claude' agent to exist")
	}
	if _, exists := agents["gpt"]; !exists {
		t.Error("Expected 'gpt' agent to exist")
	}
}

func TestManager_AddAgent(t *testing.T) {
	manager, _ := setupTestManager(t)

	newAgent := config.Agent{
		Name: "Gemini",
		Type: config.AgentTypeTmux,
		Environment: map[string]string{
			"GOOGLE_API_KEY": "test-key",
		},
		Params: &config.TmuxParams{
			Command: "gemini",
		},
	}

	if err := manager.AddAgent("gemini", newAgent); err != nil {
		t.Fatalf("Failed to add agent: %v", err)
	}

	// Verify agent was added
	agent, err := manager.GetAgent("gemini")
	if err != nil {
		t.Fatalf("Failed to get added agent: %v", err)
	}

	if agent.Name != "Gemini" {
		t.Errorf("Expected agent name 'Gemini', got '%s'", agent.Name)
	}
}

func TestManager_UpdateAgent(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Update existing agent
	updatedAgent := config.Agent{
		Name: "Claude Updated",
		Type: config.AgentTypeTmux,
		Environment: map[string]string{
			"ANTHROPIC_API_KEY": "new-key",
			"DEBUG":             "true",
		},
		Params: &config.TmuxParams{
			Command: "claude-v2",
		},
	}

	if err := manager.UpdateAgent("claude", updatedAgent); err != nil {
		t.Fatalf("Failed to update agent: %v", err)
	}

	// Verify update
	agent, err := manager.GetAgent("claude")
	if err != nil {
		t.Fatalf("Failed to get updated agent: %v", err)
	}

	if agent.Name != "Claude Updated" {
		t.Errorf("Expected updated name, got '%s'", agent.Name)
	}

	// Check updated tmux options
	params, err := agent.GetTmuxParams()
	if err != nil {
		t.Errorf("Failed to get tmux options: %v", err)
	}
	if params.Command != "claude-v2" {
		t.Errorf("Expected updated tmux command 'claude-v2', got '%s'", params.Command)
	}

	if len(agent.Environment) != 2 {
		t.Errorf("Expected 2 environment variables, got %d", len(agent.Environment))
	}

	// Try to update non-existent agent
	if err := manager.UpdateAgent("nonexistent", updatedAgent); err == nil {
		t.Error("Expected error when updating non-existent agent")
	}
}

func TestManager_RemoveAgent(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Remove existing agent
	if err := manager.RemoveAgent("gpt"); err != nil {
		t.Fatalf("Failed to remove agent: %v", err)
	}

	// Verify removal
	_, err := manager.GetAgent("gpt")
	if err == nil {
		t.Error("Expected error when getting removed agent")
	}

	// Try to remove non-existent agent
	if err := manager.RemoveAgent("nonexistent"); err == nil {
		t.Error("Expected error when removing non-existent agent")
	}
}

func TestManager_GetDefaultCommand(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Test agent with command
	cmd, err := manager.GetDefaultCommand("claude")
	if err != nil {
		t.Fatalf("Failed to get default command: %v", err)
	}
	if cmd != "claude" {
		t.Errorf("Expected command 'claude', got '%s'", cmd)
	}

	// Test agent without command (should use agent ID)
	cmd, err = manager.GetDefaultCommand("gpt")
	if err != nil {
		t.Fatalf("Failed to get default command: %v", err)
	}
	if cmd != "gpt" {
		t.Errorf("Expected command 'gpt', got '%s'", cmd)
	}

	// Test non-existent agent (should use agent ID)
	cmd, err = manager.GetDefaultCommand("unknown")
	if err != nil {
		t.Fatalf("Failed to get default command: %v", err)
	}
	if cmd != "unknown" {
		t.Errorf("Expected command 'unknown', got '%s'", cmd)
	}
}

func TestManager_GetEnvironment(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Test agent with environment
	env, err := manager.GetEnvironment("claude")
	if err != nil {
		t.Fatalf("Failed to get environment: %v", err)
	}
	if len(env) != 1 {
		t.Errorf("Expected 1 environment variable, got %d", len(env))
	}
	if env["ANTHROPIC_API_KEY"] != "test-key" {
		t.Error("Expected environment variable not found")
	}

	// Test agent without environment
	env, err = manager.GetEnvironment("gpt")
	if err != nil {
		t.Fatalf("Failed to get environment: %v", err)
	}
	if len(env) != 0 {
		t.Errorf("Expected empty environment for agent without environment, got %d", len(env))
	}

	// Test non-existent agent
	env, err = manager.GetEnvironment("unknown")
	if err != nil {
		t.Fatalf("Failed to get environment: %v", err)
	}
	if env != nil {
		t.Error("Expected nil environment for non-existent agent")
	}
}
