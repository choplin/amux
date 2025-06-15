package session

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/tests/helpers"
)

func TestCreateAutoWorkspace(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	// Generate a test session ID
	sessionID := session.GenerateID()

	// Test auto workspace creation with default name and description
	ws, err := createAutoWorkspace(context.Background(), wsManager, sessionID, "", "")
	require.NoError(t, err)
	assert.NotNil(t, ws)

	// Expected name format: session-{first-8-chars-of-uuid}
	expectedName := fmt.Sprintf("session-%s", sessionID.Short())
	assert.Equal(t, expectedName, ws.Name)
	assert.Contains(t, ws.Description, "Auto-created workspace for session")
	assert.Contains(t, ws.Description, sessionID.Short())
	assert.Equal(t, "main", ws.BaseBranch)
}

func TestCreateAutoWorkspaceUniqueness(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	// Create multiple workspaces with different session IDs
	sessionID1 := session.GenerateID()
	sessionID2 := session.GenerateID()

	ws1, err := createAutoWorkspace(context.Background(), wsManager, sessionID1, "", "")
	require.NoError(t, err)

	ws2, err := createAutoWorkspace(context.Background(), wsManager, sessionID2, "", "")
	require.NoError(t, err)

	// Ensure workspace names are different
	assert.NotEqual(t, ws1.Name, ws2.Name)
	assert.Equal(t, fmt.Sprintf("session-%s", sessionID1.Short()), ws1.Name)
	assert.Equal(t, fmt.Sprintf("session-%s", sessionID2.Short()), ws2.Name)
}

func TestCreateAutoWorkspaceWithCustomNameAndDescription(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	// Generate a test session ID
	sessionID := session.GenerateID()

	// Test auto workspace creation with custom name and description
	customName := "my-custom-workspace"
	customDescription := "This is a custom workspace for testing"
	ws, err := createAutoWorkspace(context.Background(), wsManager, sessionID, customName, customDescription)
	require.NoError(t, err)
	assert.NotNil(t, ws)

	// Verify custom name and description are used
	assert.Equal(t, customName, ws.Name)
	assert.Equal(t, customDescription, ws.Description)
	assert.Equal(t, "main", ws.BaseBranch)
}

func TestCreateAutoWorkspaceWithCustomNameOnly(t *testing.T) {
	// Create test repository
	repoDir := helpers.CreateTestRepo(t)

	// Initialize Amux
	configManager := config.NewManager(repoDir)
	cfg := config.DefaultConfig()
	err := configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	// Generate a test session ID
	sessionID := session.GenerateID()

	// Test auto workspace creation with custom name only
	customName := "custom-name-only"
	ws, err := createAutoWorkspace(context.Background(), wsManager, sessionID, customName, "")
	require.NoError(t, err)
	assert.NotNil(t, ws)

	// Verify custom name is used and default description is generated
	assert.Equal(t, customName, ws.Name)
	assert.Contains(t, ws.Description, "Auto-created workspace for session")
	assert.Contains(t, ws.Description, sessionID.Short())
}
