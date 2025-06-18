package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Config represents the main Amux configuration
type Config struct {
	Version string           `yaml:"version"`
	MCP     MCPConfig        `yaml:"mcp"`
	Agents  map[string]Agent `yaml:"agents"`
}

// MCPConfig represents MCP server configuration
type MCPConfig struct {
	Transport TransportConfig `yaml:"transport"`
}

// TransportConfig represents MCP transport configuration
type TransportConfig struct {
	Type string     `yaml:"type"`
	HTTP HTTPConfig `yaml:"http,omitempty"`
}

// HTTPConfig represents HTTP transport configuration
type HTTPConfig struct {
	Port int        `yaml:"port"`
	Auth AuthConfig `yaml:"auth,omitempty"`
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Type   string `yaml:"type"`
	Bearer string `yaml:"bearer,omitempty"`
	Basic  struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"basic,omitempty"`
}

// AgentType represents the type of agent session backend
type AgentType string

// Supported agent types
const (
	AgentTypeTmux       AgentType = "tmux"
	AgentTypeClaudeCode AgentType = "claude-code" // Future implementation
	AgentTypeAPI        AgentType = "api"         // Future implementation
)

// Agent represents an AI agent configuration
type Agent struct {
	Name        string            `yaml:"name"`
	Type        AgentType         `yaml:"type"` // Required: agent session type
	Description string            `yaml:"description,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	WorkingDir  string            `yaml:"workingDir,omitempty"`
	Tags        []string          `yaml:"tags,omitempty"`

	// Type-specific parameters - actual type depends on Type field
	Params interface{} `yaml:"params,omitempty"`
}

// TmuxParams contains tmux-specific session parameters
type TmuxParams struct {
	Command    string `yaml:"command"`
	Shell      string `yaml:"shell,omitempty"`
	WindowName string `yaml:"windowName,omitempty"`
	Detached   bool   `yaml:"detached,omitempty"`
	AutoAttach bool   `yaml:"autoAttach,omitempty"`
}

// GetTmuxParams returns tmux parameters if this is a tmux agent
func (a *Agent) GetTmuxParams() (*TmuxParams, error) {
	if a.Type != AgentTypeTmux {
		return nil, fmt.Errorf("agent %s is not a tmux agent (type: %s)", a.Name, a.Type)
	}

	params, ok := a.Params.(*TmuxParams)
	if !ok {
		return nil, fmt.Errorf("invalid tmux parameters for agent %s", a.Name)
	}

	return params, nil
}

// UnmarshalYAML implements custom YAML unmarshaling to handle type-specific parameters
func (a *Agent) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Temporary struct for initial unmarshaling
	var raw struct {
		Name        string            `yaml:"name"`
		Type        string            `yaml:"type"`
		Description string            `yaml:"description,omitempty"`
		Environment map[string]string `yaml:"environment,omitempty"`
		WorkingDir  string            `yaml:"workingDir,omitempty"`
		Tags        []string          `yaml:"tags,omitempty"`
		Params      interface{}       `yaml:"params,omitempty"`
	}

	if err := unmarshal(&raw); err != nil {
		return err
	}

	// Copy basic fields
	*a = Agent{
		Name:        raw.Name,
		Type:        AgentType(raw.Type),
		Description: raw.Description,
		Environment: raw.Environment,
		WorkingDir:  raw.WorkingDir,
		Tags:        raw.Tags,
	}

	// Convert params to appropriate concrete type based on agent type
	if raw.Params != nil {
		switch a.Type {
		case AgentTypeTmux:
			// Convert to TmuxParams
			params := &TmuxParams{}
			if err := remarshalParams(raw.Params, params); err != nil {
				return fmt.Errorf("failed to parse tmux parameters: %w", err)
			}
			a.Params = params

		case AgentTypeClaudeCode, AgentTypeAPI:
			// Future implementation - for now, keep as-is
			a.Params = raw.Params

		default:
			// Should not happen if JSON Schema validation is working
			a.Params = raw.Params
		}
	}

	return nil
}

// remarshalParams converts interface{} to a specific type via YAML remarshaling
func remarshalParams(from, to interface{}) error {
	data, err := yaml.Marshal(from)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, to)
}

// DefaultConfig returns the default Amux configuration
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		MCP: MCPConfig{
			Transport: TransportConfig{
				Type: "stdio",
			},
		},
		Agents: map[string]Agent{
			"claude": {
				Name:        "Claude",
				Type:        AgentTypeTmux,
				Description: "Claude AI assistant for terminal-based development",
				Params: &TmuxParams{
					Command: "claude",
				},
			},
		},
	}
}
