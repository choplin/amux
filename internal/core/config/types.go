package config

// Config represents the main Amux configuration
type Config struct {
	Version string           `yaml:"version"`
	Project ProjectConfig    `yaml:"project"`
	MCP     MCPConfig        `yaml:"mcp"`
	Agents  map[string]Agent `yaml:"agents"`
}

// ProjectConfig represents project-specific configuration
type ProjectConfig struct {
	Name         string `yaml:"name"`
	Repository   string `yaml:"repository"`
	DefaultAgent string `yaml:"defaultAgent"`
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
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Endpoint string `yaml:"endpoint,omitempty"`
}

// DefaultConfig returns the default Amux configuration
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Project: ProjectConfig{
			Name:         "amux-project",
			Repository:   "",
			DefaultAgent: "claude",
		},
		MCP: MCPConfig{
			Transport: TransportConfig{
				Type: "stdio",
			},
		},
		Agents: map[string]Agent{
			"claude": {
				Name: "Claude",
				Type: "claude",
			},
		},
	}
}
