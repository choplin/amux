package app

import (
	"testing"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NewTestContainer creates a container for testing with a temporary git repository
func NewTestContainer(t *testing.T) *Container {
	t.Helper()

	// Create a test repository
	repoPath := helpers.CreateTestRepo(t)

	// Initialize config
	configManager := config.NewManager(repoPath)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	require.NoError(t, err, "Failed to save config")

	// Create container
	container, err := NewContainer(repoPath)
	require.NoError(t, err, "Failed to create container")

	return container
}

func TestNewContainer(t *testing.T) {
	// Create a test repository
	repoPath := helpers.CreateTestRepo(t)

	// Initialize config
	configManager := config.NewManager(repoPath)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create container
	container, err := NewContainer(repoPath)
	require.NoError(t, err)

	// Verify all managers are initialized
	assert.NotNil(t, container.ConfigManager)
	assert.NotNil(t, container.WorkspaceManager)
	assert.NotNil(t, container.SessionManager)
	assert.NotNil(t, container.AgentManager)
	assert.NotNil(t, container.IDMapper)
	assert.Equal(t, repoPath, container.ProjectRoot)
}

func TestNewContainer_NotInitialized(t *testing.T) {
	// Create a test repository without initializing amux
	repoPath := helpers.CreateTestRepo(t)

	// Try to create container
	container, err := NewContainer(repoPath)

	// Should fail with appropriate error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amux not initialized")
	assert.Nil(t, container)
}

func TestNewContainerWithoutInit(t *testing.T) {
	// Create a test repository without initializing amux
	repoPath := helpers.CreateTestRepo(t)

	// Create container without init check
	container := NewContainerWithoutInit(repoPath)

	// Only config manager should be initialized
	assert.NotNil(t, container.ConfigManager)
	assert.Nil(t, container.WorkspaceManager)
	assert.Nil(t, container.SessionManager)
	assert.Nil(t, container.AgentManager)
	assert.Nil(t, container.IDMapper)
	assert.Equal(t, repoPath, container.ProjectRoot)
}
