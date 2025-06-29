package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_Load(t *testing.T) {
	t.Run("empty directories", func(t *testing.T) {
		loader := NewLoader("", "")
		config, err := loader.Load()
		require.NoError(t, err)
		assert.Empty(t, config.Runtimes)
	})

	t.Run("load global config", func(t *testing.T) {
		homeDir := t.TempDir()
		amuxDir := filepath.Join(homeDir, ".amux")
		require.NoError(t, os.MkdirAll(amuxDir, 0o755))

		// Create global config
		globalConfig := `
runtimes:
  dev-local:
    type: local
    description: "Local runtime with development settings"
    defaultOptions:
      inheritEnv: true
      shell: "/bin/bash"
`
		require.NoError(t, os.WriteFile(
			filepath.Join(amuxDir, "runtimes.yaml"),
			[]byte(globalConfig),
			0o644,
		))

		loader := NewLoader(homeDir, "")
		config, err := loader.Load()
		require.NoError(t, err)

		assert.Len(t, config.Runtimes, 1)
		assert.Contains(t, config.Runtimes, "dev-local")
		assert.Equal(t, "local", config.Runtimes["dev-local"].Type)
		assert.Equal(t, "Local runtime with development settings", config.Runtimes["dev-local"].Description)
	})

	t.Run("project overrides global", func(t *testing.T) {
		homeDir := t.TempDir()
		projectDir := t.TempDir()

		// Create dirs
		require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".amux"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(projectDir, ".amux"), 0o755))

		// Global config
		globalConfig := `
runtimes:
  custom-tmux:
    type: tmux
    description: "Global tmux"
    defaultOptions:
      socketPath: "/tmp/global.sock"
`
		require.NoError(t, os.WriteFile(
			filepath.Join(homeDir, ".amux", "runtimes.yaml"),
			[]byte(globalConfig),
			0o644,
		))

		// Project config
		projectConfig := `
runtimes:
  custom-tmux:
    type: tmux
    description: "Project tmux"
    defaultOptions:
      socketPath: "/tmp/project.sock"
  project-local:
    type: local
    description: "Project-specific local runtime"
`
		require.NoError(t, os.WriteFile(
			filepath.Join(projectDir, ".amux", "runtimes.yaml"),
			[]byte(projectConfig),
			0o644,
		))

		loader := NewLoader(homeDir, projectDir)
		config, err := loader.Load()
		require.NoError(t, err)

		assert.Len(t, config.Runtimes, 2)

		// Project should override global
		assert.Equal(t, "Project tmux", config.Runtimes["custom-tmux"].Description)
		assert.Equal(t, "/tmp/project.sock", config.Runtimes["custom-tmux"].DefaultOptions["socketPath"])

		// Project-specific runtime should exist
		assert.Contains(t, config.Runtimes, "project-local")
	})
}

func TestRuntimeConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := &RuntimeConfig{
			Runtimes: map[string]RuntimeDefinition{
				"my-local": {Type: "local"},
				"my-tmux":  {Type: "tmux"},
			},
		}
		assert.NoError(t, config.Validate())
	})

	t.Run("missing type", func(t *testing.T) {
		config := &RuntimeConfig{
			Runtimes: map[string]RuntimeDefinition{
				"invalid": {Description: "No type specified"},
			},
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "type is required")
	})

	t.Run("unknown type", func(t *testing.T) {
		config := &RuntimeConfig{
			Runtimes: map[string]RuntimeDefinition{
				"invalid": {Type: "docker"},
			},
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown type")
	})
}
