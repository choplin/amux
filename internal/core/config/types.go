package config

import (
	"fmt"
	"strings"

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

// Command represents a command that can be either a string or an array of strings
type Command struct {
	// Single is set when the command is a string
	Single string
	// Array is set when the command is an array
	Array []string
}

// IsArray returns true if the command is an array
func (c Command) IsArray() bool {
	return c.Array != nil
}

// String returns the command as a string
// For array commands, it returns the first element or empty string
func (c Command) String() string {
	if c.IsArray() {
		if len(c.Array) > 0 {
			return c.Array[0]
		}
		return ""
	}
	return c.Single
}

// UnmarshalYAML implements yaml.Unmarshaler to handle both string and []string
func (c *Command) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First try to unmarshal as interface{} to check the type
	var raw interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	switch v := raw.(type) {
	case string:
		c.Single = v
		c.Array = nil
		return nil
	case []interface{}:
		// Convert []interface{} to []string
		arr := make([]string, len(v))
		for i, item := range v {
			str, ok := item.(string)
			if !ok {
				return fmt.Errorf("array element %d must be a string, got %T", i, item)
			}
			arr[i] = str
		}
		c.Array = arr
		c.Single = ""
		return nil
	case nil:
		return fmt.Errorf("command cannot be null")
	default:
		return fmt.Errorf("command must be either a string or an array of strings, got %T", v)
	}
}

// MarshalYAML implements yaml.Marshaler to output as string or array
func (c Command) MarshalYAML() (interface{}, error) {
	if c.IsArray() {
		return c.Array, nil
	}
	return c.Single, nil
}

// TmuxParams contains tmux-specific session parameters
type TmuxParams struct {
	// Command can be either string (shell execution) or []string (direct execution)
	Command    Command `yaml:"command"`
	WindowName string  `yaml:"windowName,omitempty"`
	Detached   bool    `yaml:"detached,omitempty"`
	AutoAttach bool    `yaml:"autoAttach,omitempty"`
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

// GetAgent returns the agent with the specified ID
func (c *Config) GetAgent(id string) (*Agent, error) {
	agent, exists := c.Agents[id]
	if !exists {
		return nil, fmt.Errorf("agent %q not found", id)
	}
	return &agent, nil
}

// GetAgentType returns the type of the specified agent
func (c *Config) GetAgentType(id string) (AgentType, error) {
	agent, err := c.GetAgent(id)
	if err != nil {
		return "", err
	}
	return agent.Type, nil
}

// TmuxAgent represents a type-safe tmux agent configuration
type TmuxAgent struct {
	*Agent
	Params *TmuxParams
}

// GetTmuxAgent returns the agent as a TmuxAgent if it's a tmux type
func (c *Config) GetTmuxAgent(id string) (*TmuxAgent, error) {
	agent, err := c.GetAgent(id)
	if err != nil {
		return nil, err
	}

	if agent.Type != AgentTypeTmux {
		return nil, fmt.Errorf("agent %q is not a tmux agent (type: %s)", id, agent.Type)
	}

	params, err := agent.GetTmuxParams()
	if err != nil {
		return nil, err
	}

	return &TmuxAgent{
		Agent:  agent,
		Params: params,
	}, nil
}

// GetCommand returns the command as a string
func (t *TmuxAgent) GetCommand() string {
	if t.Params.Command.IsArray() {
		return strings.Join(t.Params.Command.Array, " ")
	}
	return t.Params.Command.Single
}

// GetEnvironment returns the environment variables
func (t *TmuxAgent) GetEnvironment() map[string]string {
	if t.Environment == nil {
		return make(map[string]string)
	}
	return t.Environment
}

// ShouldAutoAttach returns whether to auto-attach to the session
func (t *TmuxAgent) ShouldAutoAttach() bool {
	return t.Params.AutoAttach
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
					Command: Command{Single: "claude"},
				},
			},
		},
	}
}
