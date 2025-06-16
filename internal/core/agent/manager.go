// Package agent provides AI agent configuration and management.
package agent

import (
	"fmt"

	"github.com/aki/amux/internal/core/config"
)

// Manager manages agent configurations
type Manager struct {
	configManager *config.Manager
}

// NewManager creates a new agent manager
func NewManager(configManager *config.Manager) *Manager {
	return &Manager{
		configManager: configManager,
	}
}

// GetAgent retrieves an agent configuration by ID
func (m *Manager) GetAgent(agentID string) (*config.Agent, error) {
	cfg, err := m.configManager.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	agent, exists := cfg.Agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent '%s' not found in configuration", agentID)
	}

	return &agent, nil
}

// ListAgents returns all configured agents
func (m *Manager) ListAgents() (map[string]config.Agent, error) {
	cfg, err := m.configManager.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg.Agents, nil
}

// AddAgent adds a new agent configuration
func (m *Manager) AddAgent(id string, agent config.Agent) error {
	cfg, err := m.configManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Agents == nil {
		cfg.Agents = make(map[string]config.Agent)
	}

	cfg.Agents[id] = agent

	if err := m.configManager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// UpdateAgent updates an existing agent configuration
func (m *Manager) UpdateAgent(id string, agent config.Agent) error {
	cfg, err := m.configManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if _, exists := cfg.Agents[id]; !exists {
		return fmt.Errorf("agent '%s' not found", id)
	}

	cfg.Agents[id] = agent

	if err := m.configManager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// RemoveAgent removes an agent configuration
func (m *Manager) RemoveAgent(id string) error {
	cfg, err := m.configManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if _, exists := cfg.Agents[id]; !exists {
		return fmt.Errorf("agent '%s' not found", id)
	}

	delete(cfg.Agents, id)

	if err := m.configManager.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// GetDefaultCommand returns the command to run for an agent
// Falls back to agent ID if no command is specified
func (m *Manager) GetDefaultCommand(agentID string) (string, error) {
	agent, err := m.GetAgent(agentID)
	if err != nil {
		// If agent not found, use the agent ID as command
		return agentID, nil //nolint:nilerr // Fallback to agent ID if not configured
	}

	// Get command based on agent type
	switch agent.Type {
	case "tmux":
		if agent.Tmux != nil && agent.Tmux.Command != "" {
			return agent.Tmux.Command, nil
		}
		// Future: handle other types
	}

	// Default to agent ID as command
	return agentID, nil
}

// GetEnvironment returns the environment variables for an agent
func (m *Manager) GetEnvironment(agentID string) (map[string]string, error) {
	agent, err := m.GetAgent(agentID)
	if err != nil {
		// If agent not found, return empty environment
		return nil, nil //nolint:nilerr // Return empty env if agent not configured
	}

	return agent.Environment, nil
}
