package commands

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/workspace"
	"github.com/aki/amux/internal/tests/helpers"
)

func TestCreateAutoWorkspace(t *testing.T) {
	tests := []struct {
		name            string
		existingWS      []string
		expectedName    string
		expectedPattern string
	}{
		{
			name:         "first auto workspace",
			existingWS:   []string{},
			expectedName: "auto-1",
		},
		{
			name:         "second auto workspace",
			existingWS:   []string{"auto-1"},
			expectedName: "auto-2",
		},
		{
			name:         "with gap in numbering",
			existingWS:   []string{"auto-1", "auto-3"},
			expectedName: "auto-2",
		},
		{
			name:         "multiple existing",
			existingWS:   []string{"auto-1", "auto-2", "auto-3"},
			expectedName: "auto-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test repository
			repoDir := helpers.CreateTestRepo(t)
			defer os.RemoveAll(repoDir)

			// Initialize Amux
			configManager := config.NewManager(repoDir)
			cfg := config.DefaultConfig()
			err := configManager.Save(cfg)
			require.NoError(t, err)

			// Create workspace manager
			wsManager, err := workspace.NewManager(configManager)
			require.NoError(t, err)

			// Create existing workspaces
			for _, name := range tt.existingWS {
				opts := workspace.CreateOptions{
					Name:        name,
					Description: "Test workspace",
					BaseBranch:  "main",
				}
				_, err := wsManager.Create(opts)
				require.NoError(t, err)
			}

			// Test auto workspace creation
			ws, err := createAutoWorkspace(wsManager)
			require.NoError(t, err)
			assert.NotNil(t, ws)
			assert.Equal(t, tt.expectedName, ws.Name)
			assert.Equal(t, "Auto-created workspace", ws.Description)
			assert.Equal(t, "main", ws.BaseBranch)
		})
	}
}
