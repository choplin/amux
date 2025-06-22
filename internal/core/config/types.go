package config

import (
	"fmt"
	"strconv"
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
	AgentTypeBlocking   AgentType = "blocking"    // Direct process execution
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

// BlockingParams contains blocking session-specific parameters
type BlockingParams struct {
	Command string   `yaml:"command"`        // Command to execute
	Args    []string `yaml:"args,omitempty"` // Command arguments
	Output  struct {
		Mode       string `yaml:"mode,omitempty"`       // buffer, file, circular
		BufferSize string `yaml:"bufferSize,omitempty"` // e.g., "10MB", "1GB"
		FilePath   string `yaml:"filePath,omitempty"`   // for file mode
	} `yaml:"output,omitempty"`
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

// GetBlockingParams returns blocking parameters if this is a blocking agent
func (a *Agent) GetBlockingParams() (*BlockingParams, error) {
	if a.Type != AgentTypeBlocking {
		return nil, fmt.Errorf("agent %s is not a blocking agent (type: %s)", a.Name, a.Type)
	}

	params, ok := a.Params.(*BlockingParams)
	if !ok {
		return nil, fmt.Errorf("invalid blocking parameters for agent %s", a.Name)
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

		case AgentTypeBlocking:
			// Convert to BlockingParams
			params := &BlockingParams{}
			if err := remarshalParams(raw.Params, params); err != nil {
				return fmt.Errorf("failed to parse blocking parameters: %w", err)
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

// ParseBufferSize parses a human-readable buffer size string (e.g., "10MB", "1GB") into bytes
func ParseBufferSize(size string) (int64, error) {
	if size == "" {
		return 0, nil
	}

	size = strings.TrimSpace(size)

	// Extract numeric part and unit
	var numStr string
	var unit string

	for i, r := range size {
		if (r >= '0' && r <= '9') || r == '.' {
			continue
		}
		numStr = size[:i]
		unit = strings.ToUpper(size[i:])
		break
	}

	if numStr == "" {
		return 0, fmt.Errorf("invalid buffer size: %s", size)
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid buffer size: %s", size)
	}

	// Convert to bytes based on unit
	var multiplier float64 = 1
	switch unit {
	case "B", "":
		multiplier = 1
	case "K", "KB":
		multiplier = 1024
	case "M", "MB":
		multiplier = 1024 * 1024
	case "G", "GB":
		multiplier = 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown buffer size unit: %s", unit)
	}

	return int64(num * multiplier), nil
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
