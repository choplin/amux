// Package config provides configuration management for Amux projects.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// AmuxDir is the directory name for Amux metadata
	AmuxDir = ".amux"
	// ConfigFile is the filename for the Amux configuration
	ConfigFile = "config.yaml"
)

// Manager handles Amux configuration
type Manager struct {
	projectRoot string
	configPath  string
}

// NewManager creates a new configuration manager
func NewManager(projectRoot string) *Manager {
	return &Manager{
		projectRoot: projectRoot,
		configPath:  filepath.Join(projectRoot, AmuxDir, ConfigFile),
	}
}

// Load reads the configuration from disk
func (m *Manager) Load() (*Config, error) {
	// Use JSON schema validation
	config, err := LoadWithValidation(m.configPath)
	if err != nil {
		// Customize error message for missing file
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("amux not initialized. Run 'amux init' first")
		}
		return nil, err
	}

	// Apply defaults after validation
	applyDefaults(config)

	return config, nil
}

// Save writes the configuration to disk
func (m *Manager) Save(config *Config) error {
	// Ensure the .amux directory exists
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// IsInitialized checks if Amux has been initialized in the project
func (m *Manager) IsInitialized() bool {
	_, err := os.Stat(m.configPath)
	return err == nil
}

// GetProjectRoot returns the project root directory
func (m *Manager) GetProjectRoot() string {
	return m.projectRoot
}

// GetAmuxDir returns the .amux directory path
func (m *Manager) GetAmuxDir() string {
	return filepath.Join(m.projectRoot, AmuxDir)
}

// GetWorkspacesDir returns the workspaces directory path
func (m *Manager) GetWorkspacesDir() string {
	return filepath.Join(m.projectRoot, AmuxDir, "workspaces")
}

// GetAgent retrieves an agent configuration by ID
func (m *Manager) GetAgent(agentID string) (*Agent, error) {
	cfg, err := m.Load()
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
func (m *Manager) ListAgents() (map[string]Agent, error) {
	cfg, err := m.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg.Agents, nil
}

// AddAgent adds a new agent configuration
func (m *Manager) AddAgent(id string, agent Agent) error {
	cfg, err := m.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Agents == nil {
		cfg.Agents = make(map[string]Agent)
	}

	cfg.Agents[id] = agent

	if err := m.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// UpdateAgent updates an existing agent configuration
func (m *Manager) UpdateAgent(id string, agent Agent) error {
	cfg, err := m.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if _, exists := cfg.Agents[id]; !exists {
		return fmt.Errorf("agent '%s' not found", id)
	}

	cfg.Agents[id] = agent

	if err := m.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// RemoveAgent removes an agent configuration
func (m *Manager) RemoveAgent(id string) error {
	cfg, err := m.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if _, exists := cfg.Agents[id]; !exists {
		return fmt.Errorf("agent '%s' not found", id)
	}

	delete(cfg.Agents, id)

	if err := m.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// LoadProjectConfig loads the project configuration from the .amux directory
func LoadProjectConfig(projectRoot string) (*Config, error) {
	path := filepath.Join(projectRoot, AmuxDir, ConfigFile)
	config, err := LoadWithValidation(path)
	if err != nil {
		// Customize error message for missing file
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("amux not initialized. Run 'amux init' first")
		}
		return nil, err
	}

	// Apply defaults after validation
	applyDefaults(config)

	return config, nil
}

// FindProjectRoot searches for the project root by looking for .amux directory
func FindProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for .amux
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, AmuxDir, ConfigFile)); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not in an Amux project (no .amux directory found)")
}

// GetConfigPath returns the configuration file path
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// applyDefaults applies default values to the configuration
func applyDefaults(cfg *Config) {
	// Apply default version if empty
	if cfg.Version == "" {
		cfg.Version = "1.0"
	}

	// Apply default MCP transport if not set
	if cfg.MCP.Transport.Type == "" {
		cfg.MCP.Transport.Type = "stdio"
	}
}
