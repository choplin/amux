package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Loader handles loading runtime configurations
type Loader struct {
	homeDir    string
	projectDir string
}

// NewLoader creates a new configuration loader
func NewLoader(homeDir, projectDir string) *Loader {
	return &Loader{
		homeDir:    homeDir,
		projectDir: projectDir,
	}
}

// Load loads runtime configurations from both global and project-specific files
func (l *Loader) Load() (*RuntimeConfig, error) {
	config := &RuntimeConfig{
		Runtimes: make(map[string]RuntimeDefinition),
	}

	// Load global config first
	if l.homeDir != "" {
		globalPath := filepath.Join(l.homeDir, ".amux", "runtimes.yaml")
		if err := l.loadFile(globalPath, config); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load global config: %w", err)
		}
	}

	// Load project config (overrides global)
	if l.projectDir != "" {
		projectPath := filepath.Join(l.projectDir, ".amux", "runtimes.yaml")
		if err := l.loadFile(projectPath, config); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load project config: %w", err)
		}
	}

	return config, nil
}

// loadFile loads a single configuration file
func (l *Loader) loadFile(path string, config *RuntimeConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var fileConfig RuntimeConfig
	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}

	// Merge runtimes
	for name, def := range fileConfig.Runtimes {
		config.Runtimes[name] = def
	}

	return nil
}

// Validate validates the loaded configuration
func (c *RuntimeConfig) Validate() error {
	for name, def := range c.Runtimes {
		if def.Type == "" {
			return fmt.Errorf("runtime %q: type is required", name)
		}

		// Validate type is one of the built-in types
		switch def.Type {
		case "local", "tmux":
			// Valid types
		default:
			return fmt.Errorf("runtime %q: unknown type %q", name, def.Type)
		}
	}

	return nil
}
