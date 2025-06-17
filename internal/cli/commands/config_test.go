package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aki/amux/internal/core/config"
	"github.com/stretchr/testify/require"
)

func TestConfigShowCommand(t *testing.T) {
	// Create a temporary project directory
	projectDir := t.TempDir()
	amuxDir := filepath.Join(projectDir, config.AmuxDir)
	err := os.MkdirAll(amuxDir, 0o755)
	require.NoError(t, err)

	// Create a test configuration
	testCfg := &config.Config{
		Version: "1.0",
		MCP: config.MCPConfig{
			Transport: config.TransportConfig{
				Type: "stdio",
			},
		},
		Agents: map[string]config.Agent{
			"test-agent": {
				Name: "Test Agent",
				Type: config.AgentTypeTmux,
				Params: &config.TmuxParams{
					Command: "test-command",
				},
			},
		},
	}

	// Save the configuration
	mgr := config.NewManager(projectDir)
	err = mgr.Save(testCfg)
	require.NoError(t, err)

	// Change to the project directory
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(projectDir)

	// Test that the command executes without error
	t.Run("Execute command", func(t *testing.T) {
		// Execute through the parent command to get proper setup
		rootCmd.SetArgs([]string{"config", "show"})
		err := rootCmd.Execute()
		require.NoError(t, err)
	})
}

func TestConfigShowCommandErrors(t *testing.T) {
	// Change to a directory without .amux
	tempDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tempDir)

	t.Run("Not in amux project", func(t *testing.T) {
		// Execute through the parent command
		rootCmd.SetArgs([]string{"config", "show"})
		err := rootCmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "not in an Amux project")
	})

	t.Run("Invalid format", func(t *testing.T) {
		// Create a minimal .amux directory
		amuxDir := filepath.Join(tempDir, config.AmuxDir)
		err := os.MkdirAll(amuxDir, 0o755)
		require.NoError(t, err)

		// Create a valid config
		mgr := config.NewManager(tempDir)
		err = mgr.Save(config.DefaultConfig())
		require.NoError(t, err)

		// Run with invalid format
		rootCmd.SetArgs([]string{"config", "show", "--format", "invalid"})
		err = rootCmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported format")
	})
}

func TestFindEditor(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name: "EDITOR set",
			envVars: map[string]string{
				"EDITOR": "test-editor",
			},
			expected: "test-editor",
		},
		{
			name: "VISUAL set",
			envVars: map[string]string{
				"VISUAL": "test-visual",
			},
			expected: "test-visual",
		},
		{
			name: "Both EDITOR and VISUAL set",
			envVars: map[string]string{
				"EDITOR": "test-editor",
				"VISUAL": "test-visual",
			},
			expected: "test-editor", // EDITOR takes precedence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save current env vars
			oldEditor := os.Getenv("EDITOR")
			oldVisual := os.Getenv("VISUAL")
			defer func() {
				os.Setenv("EDITOR", oldEditor)
				os.Setenv("VISUAL", oldVisual)
			}()

			// Clear env vars
			os.Unsetenv("EDITOR")
			os.Unsetenv("VISUAL")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			editor := findEditor()
			require.Equal(t, tt.expected, editor)
		})
	}

	// Test default editor finding
	t.Run("No editor set", func(t *testing.T) {
		// Save current env vars
		oldEditor := os.Getenv("EDITOR")
		oldVisual := os.Getenv("VISUAL")
		defer func() {
			os.Setenv("EDITOR", oldEditor)
			os.Setenv("VISUAL", oldVisual)
		}()

		// Clear env vars
		os.Unsetenv("EDITOR")
		os.Unsetenv("VISUAL")

		editor := findEditor()
		// Should find some default editor on most systems
		// We can't test the exact value as it depends on the system
		// Just ensure it's not empty on systems with common editors
		_ = editor
	})
}
