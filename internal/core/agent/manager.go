// Package agent provides AI agent configuration and management.
package agent

import (
	"fmt"

	"github.com/aki/amux/internal/core/config"
)

const (
	// DefaultShell is the fallback shell used when no command or shell is configured
	DefaultShell = "bash"

	// DefaultAgentID is the default agent identifier when none is specified
	DefaultAgentID = "default"
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
// Returns error if no command is configured
func (m *Manager) GetDefaultCommand(agentID string) (string, error) {
	agent, err := m.GetAgent(agentID)
	if err != nil {
		return "", fmt.Errorf("agent %q not found", agentID)
	}

	// Get command based on agent type
	switch agent.Type {
	case config.AgentTypeTmux:
		params, err := agent.GetTmuxParams()
		if err != nil {
			return "", fmt.Errorf("failed to get tmux params: %w", err)
		}
		if params.Command != "" {
			return params.Command, nil
		}
		// TODO(#218): This fallback logic will be removed when we refactor to follow
		// the tmux command execution model. The shell field will be obsoleted and
		// command will handle both string (shell execution) and array (direct execution).
		// If no command is specified, use the shell if configured
		if params.Shell != "" {
			return params.Shell, nil
		}
		// Fall back to the default shell
		return DefaultShell, nil
	case config.AgentTypeClaudeCode, config.AgentTypeAPI:
		// Future: handle other types
		return "", fmt.Errorf("agent type %q not yet supported", agent.Type)
	default:
		return "", fmt.Errorf("unknown agent type %q", agent.Type)
	}
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
