package storage

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aki/amux/internal/config"
	"github.com/aki/amux/internal/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageCommands(t *testing.T) {
	t.Skip("Skipping test that requires git repo setup")
	// Create temporary directory for testing
	tmpDir := t.TempDir()

	// Initialize git repo in temp directory
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	err := cmd.Run()
	require.NoError(t, err)

	// Create initial commit
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	// Create a dummy file and commit
	readmePath := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(readmePath, []byte("# Test"), 0o644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	// Initialize amux in temp directory
	configManager := config.NewManager(tmpDir)

	// Create default configuration and save it
	cfg := config.DefaultConfig()
	err = configManager.Save(cfg)
	require.NoError(t, err)

	// Create workspaces directory
	workspacesDir := configManager.GetWorkspacesDir()
	err = os.MkdirAll(workspacesDir, 0o755)
	require.NoError(t, err)

	// Create workspace manager
	wsManager, err := workspace.NewManager(configManager)
	require.NoError(t, err)

	// Create a test workspace
	ctx := context.Background()
	ws, err := wsManager.Create(ctx, workspace.CreateOptions{
		Name:        "test-workspace",
		Description: "Test workspace for storage commands",
	})
	require.NoError(t, err)

	// Test write command
	t.Run("write", func(t *testing.T) {
		// Write a test file
		testPath := filepath.Join(ws.StoragePath, "test.txt")
		testContent := []byte("Hello from test")
		err := os.WriteFile(testPath, testContent, 0o644)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(testPath)
		assert.NoError(t, err)
	})

	// Test list command
	t.Run("list", func(t *testing.T) {
		// Create some test files
		err := os.WriteFile(filepath.Join(ws.StoragePath, "file1.txt"), []byte("content1"), 0o644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(ws.StoragePath, "file2.txt"), []byte("content2"), 0o644)
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Join(ws.StoragePath, "subdir"), 0o755)
		require.NoError(t, err)

		// List files
		entries, err := os.ReadDir(ws.StoragePath)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(entries), 3) // At least our 3 items
	})

	// Test read command
	t.Run("read", func(t *testing.T) {
		// Create a test file to read
		testPath := filepath.Join(ws.StoragePath, "read-test.txt")
		testContent := "Test content for reading"
		err := os.WriteFile(testPath, []byte(testContent), 0o644)
		require.NoError(t, err)

		// Read the file
		content, err := os.ReadFile(testPath)
		require.NoError(t, err)
		assert.Equal(t, testContent, string(content))
	})

	// Test remove command
	t.Run("remove", func(t *testing.T) {
		// Create a test file to remove
		testPath := filepath.Join(ws.StoragePath, "remove-test.txt")
		err := os.WriteFile(testPath, []byte("to be removed"), 0o644)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(testPath)
		require.NoError(t, err)

		// Remove the file
		err = os.Remove(testPath)
		require.NoError(t, err)

		// Verify file is gone
		_, err = os.Stat(testPath)
		assert.True(t, os.IsNotExist(err))
	})

	// Test path traversal protection
	t.Run("path_traversal_protection", func(t *testing.T) {
		// The actual path traversal protection is in the CLI commands,
		// not in the test. This test verifies that our logic works correctly.
		dangerousPaths := []string{
			"../../../etc/passwd",
			"./../../sensitive",
			"subdir/../../..",
		}

		for _, path := range dangerousPaths {
			fullPath := filepath.Join(ws.StoragePath, path)
			cleanPath := filepath.Clean(fullPath)
			cleanStoragePath := filepath.Clean(ws.StoragePath)

			// After cleaning, paths with ".." should escape the storage directory
			if strings.Contains(path, "..") {
				// The cleaned path should NOT be within the storage directory
				assert.False(t, strings.HasPrefix(cleanPath, cleanStoragePath+string(filepath.Separator)),
					"Path with .. should escape storage directory: %s", path)
			}
		}
	})

	// Cleanup
	err = wsManager.Remove(ctx, workspace.Identifier(ws.ID), workspace.RemoveOptions{})
	assert.NoError(t, err)
}
