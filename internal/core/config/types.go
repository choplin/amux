package config

import (
	"fmt"
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

// Agent represents an AI agent configuration
type Agent struct {
	Name           string            `yaml:"name"`
	Runtime        string            `yaml:"runtime"` // Runtime type ("local", "tmux")
	Description    string            `yaml:"description,omitempty"`
	Environment    map[string]string `yaml:"environment,omitempty"`
	WorkingDir     string            `yaml:"workingDir,omitempty"`
	Tags           []string          `yaml:"tags,omitempty"`
	RuntimeOptions interface{}       `yaml:"runtimeOptions,omitempty"` // Runtime-specific options
	Command        []string          `yaml:"command,omitempty"`        // Command to execute
}

// GetRuntimeType returns the runtime type for this agent
func (a *Agent) GetRuntimeType() string {
	return a.Runtime
}

// GetRuntimeOptions returns the runtime options for this agent
func (a *Agent) GetRuntimeOptions() interface{} {
	return a.RuntimeOptions
}

// GetCommand returns the command to execute
func (a *Agent) GetCommand() []string {
	return a.Command
}

// GetAgent returns the agent with the specified ID
func (c *Config) GetAgent(id string) (*Agent, error) {
	agent, exists := c.Agents[id]
	if !exists {
		return nil, fmt.Errorf("agent %q not found", id)
	}
	return &agent, nil
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
				Runtime:     "tmux",
				Description: "Claude AI assistant for terminal-based development",
				Command:     []string{"claude"},
			},
		},
	}
}
