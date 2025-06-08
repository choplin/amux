package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	AgentCaveDir = ".agentcave"
	ConfigFile   = "config.yaml"
)

// Manager handles AgentCave configuration
type Manager struct {
	projectRoot string
	configPath  string
}

// NewManager creates a new configuration manager
func NewManager(projectRoot string) *Manager {
	return &Manager{
		projectRoot: projectRoot,
		configPath:  filepath.Join(projectRoot, AgentCaveDir, ConfigFile),
	}
}

// Load reads the configuration from disk
func (m *Manager) Load() (*Config, error) {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("AgentCave not initialized. Run 'agentcave init' first")
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// Save writes the configuration to disk
func (m *Manager) Save(config *Config) error {
	// Ensure the .agentcave directory exists
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// IsInitialized checks if AgentCave has been initialized in the project
func (m *Manager) IsInitialized() bool {
	_, err := os.Stat(m.configPath)
	return err == nil
}

// GetProjectRoot returns the project root directory
func (m *Manager) GetProjectRoot() string {
	return m.projectRoot
}

// GetAgentCaveDir returns the .agentcave directory path
func (m *Manager) GetAgentCaveDir() string {
	return filepath.Join(m.projectRoot, AgentCaveDir)
}

// GetWorkspacesDir returns the workspaces directory path
func (m *Manager) GetWorkspacesDir() string {
	return filepath.Join(m.projectRoot, AgentCaveDir, "workspaces")
}

// FindProjectRoot searches for the project root by looking for .agentcave directory
func FindProjectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for .agentcave
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, AgentCaveDir, ConfigFile)); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("not in an AgentCave project (no .agentcave directory found)")
}