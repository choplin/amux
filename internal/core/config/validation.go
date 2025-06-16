package config

import (
	"fmt"
)

// ValidateConfig validates the entire configuration
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Validate agents
	for id, agent := range config.Agents {
		if err := ValidateAgent(id, &agent); err != nil {
			return fmt.Errorf("invalid agent '%s': %w", id, err)
		}
	}

	return nil
}

// ValidateAgent validates an individual agent configuration
func ValidateAgent(id string, agent *Agent) error {
	if agent == nil {
		return fmt.Errorf("agent is nil")
	}

	// Name is required
	if agent.Name == "" {
		return fmt.Errorf("name is required")
	}

	// Type is required
	if agent.Type == "" {
		return fmt.Errorf("type is required")
	}

	// Validate type-specific configuration
	switch agent.Type {
	case "tmux":
		if agent.Tmux == nil {
			return fmt.Errorf("tmux configuration is required for type 'tmux'")
		}
		if err := ValidateTmuxConfig(agent.Tmux); err != nil {
			return fmt.Errorf("invalid tmux configuration: %w", err)
		}
	// Future: case "claude-code", "api", "lsp", etc.
	default:
		return fmt.Errorf("unsupported agent type: %s", agent.Type)
	}

	return nil
}

// ValidateTmuxConfig validates tmux-specific configuration
func ValidateTmuxConfig(config *TmuxConfig) error {
	if config == nil {
		return fmt.Errorf("tmux config is nil")
	}

	// Command is required for tmux sessions
	if config.Command == "" {
		return fmt.Errorf("command is required")
	}

	return nil
}
