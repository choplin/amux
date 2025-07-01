package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aki/amux/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubdirectoryExecution tests that session commands work from subdirectories
func TestSubdirectoryExecution(t *testing.T) {
	// Create a temporary project directory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "myproject")
	subDir := filepath.Join(projectDir, "src", "components")

	// Create directories
	err := os.MkdirAll(subDir, 0o755)
	require.NoError(t, err)

	// Save current directory to restore later
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Initialize amux project
	err = os.Chdir(projectDir)
	require.NoError(t, err)

	// Create amux config directory (simulating 'amux init')
	amuxDir := filepath.Join(projectDir, ".amux")
	err = os.MkdirAll(amuxDir, 0o755)
	require.NoError(t, err)

	// Create config file
	configPath := filepath.Join(amuxDir, "config.yaml")
	configContent := `version: 1
project:
  name: myproject
  root: ` + projectDir + `
`
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Test FindProjectRoot from subdirectory
	err = os.Chdir(subDir)
	require.NoError(t, err)

	foundRoot, err := config.FindProjectRoot()
	assert.NoError(t, err)
	// Use filepath.EvalSymlinks to resolve any symlinks (handles /private prefix on macOS)
	expectedDir, _ := filepath.EvalSymlinks(projectDir)
	actualDir, _ := filepath.EvalSymlinks(foundRoot)
	assert.Equal(t, expectedDir, actualDir)

	// Test setupManagers from subdirectory
	configMgr, sessionMgr, err := setupManagers()
	assert.NoError(t, err)
	assert.NotNil(t, configMgr)
	assert.NotNil(t, sessionMgr)
	// Use filepath.EvalSymlinks to compare paths
	actualRoot, _ := filepath.EvalSymlinks(configMgr.GetProjectRoot())
	assert.Equal(t, expectedDir, actualRoot)
}

// TestSetupManagersErrors tests error handling in setupManagers
func TestSetupManagersErrors(t *testing.T) {
	// Save current directory to restore later
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Test from directory without amux project
	tmpDir := t.TempDir()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	configMgr, sessionMgr, err := setupManagers()
	assert.Error(t, err)
	assert.Nil(t, configMgr)
	assert.Nil(t, sessionMgr)
	assert.Contains(t, err.Error(), "not in an amux project")
	assert.Contains(t, err.Error(), tmpDir)
}
