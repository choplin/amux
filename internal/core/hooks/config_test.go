package hooks_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/core/hooks"
)

func TestLoadConfig(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	t.Run("returns empty config when file doesn't exist", func(t *testing.T) {
		config, err := hooks.LoadConfig(tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Empty(t, config.Hooks)
	})

	t.Run("loads valid config", func(t *testing.T) {
		// Create config file
		configPath := filepath.Join(tmpDir, hooks.HooksConfigFile)
		configContent := `
hooks:
  workspace_create:
    - name: "Test hook"
      command: "echo test"
      timeout: "1m"
      on_error: "fail"
`
		err := os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		// Load config
		config, err := hooks.LoadConfig(tmpDir)
		require.NoError(t, err)
		assert.NotNil(t, config)

		// Check hooks
		createHooks := config.GetHooksForEvent(hooks.EventWorkspaceCreate)
		require.Len(t, createHooks, 1)
		assert.Equal(t, "Test hook", createHooks[0].Name)
		assert.Equal(t, "echo test", createHooks[0].Command)
		assert.Equal(t, "1m", createHooks[0].Timeout)
		assert.Equal(t, hooks.ErrorStrategyFail, createHooks[0].OnError)
	})

	t.Run("applies defaults", func(t *testing.T) {
		// Create config file without optional fields
		configPath := filepath.Join(tmpDir, hooks.HooksConfigFile)
		configContent := `
hooks:
  workspace_create:
    - name: "Test hook"
      command: "echo test"
`
		err := os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		// Load config
		config, err := hooks.LoadConfig(tmpDir)
		require.NoError(t, err)

		// Check defaults
		createHooks := config.GetHooksForEvent(hooks.EventWorkspaceCreate)
		require.Len(t, createHooks, 1)
		assert.Equal(t, "5m", createHooks[0].Timeout)
		assert.Equal(t, hooks.ErrorStrategyWarn, createHooks[0].OnError)
	})
}

func TestTrustMechanism(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a config
	config := &hooks.Config{
		Hooks: map[string][]hooks.Hook{
			string(hooks.EventWorkspaceCreate): {
				{
					Name:    "Test",
					Command: "echo test",
				},
			},
		},
	}

	// Save config
	err := hooks.SaveConfig(tmpDir, config)
	require.NoError(t, err)

	t.Run("not trusted initially", func(t *testing.T) {
		trusted, err := hooks.IsTrusted(tmpDir, config)
		require.NoError(t, err)
		assert.False(t, trusted)
	})

	t.Run("trusted after saving trust info", func(t *testing.T) {
		// Calculate hash
		hash, err := hooks.CalculateConfigHash(config)
		require.NoError(t, err)

		// Save trust info
		trust := &hooks.TrustInfo{
			Hash:      hash,
			TrustedBy: "test",
		}
		err = hooks.SaveTrustInfo(tmpDir, trust)
		require.NoError(t, err)

		// Check trust
		trusted, err := hooks.IsTrusted(tmpDir, config)
		require.NoError(t, err)
		assert.True(t, trusted)
	})

	t.Run("not trusted after config change", func(t *testing.T) {
		// Modify config
		config.Hooks[string(hooks.EventWorkspaceCreate)] = append(
			config.Hooks[string(hooks.EventWorkspaceCreate)],
			hooks.Hook{
				Name:    "New hook",
				Command: "echo new",
			},
		)

		// Check trust
		trusted, err := hooks.IsTrusted(tmpDir, config)
		require.NoError(t, err)
		assert.False(t, trusted)
	})
}
