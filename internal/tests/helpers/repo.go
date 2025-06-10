package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// CreateTestRepo creates a temporary git repository for testing
func CreateTestRepo(t *testing.T) string {
	t.Helper()

	// Store original environment and restore after test
	origGitDir := os.Getenv("GIT_DIR")
	origGitWorkTree := os.Getenv("GIT_WORK_TREE")
	origGitIndexFile := os.Getenv("GIT_INDEX_FILE")

	// Clear git environment variables for test isolation
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("GIT_INDEX_FILE")

	// Restore original environment after test
	t.Cleanup(func() {
		if origGitDir != "" {
			os.Setenv("GIT_DIR", origGitDir)
		}
		if origGitWorkTree != "" {
			os.Setenv("GIT_WORK_TREE", origGitWorkTree)
		}
		if origGitIndexFile != "" {
			os.Setenv("GIT_INDEX_FILE", origGitIndexFile)
		}
	})

	// Create temporary directory in system temp, not in current directory
	// This ensures we're not inside any existing git repository
	systemTmp := os.TempDir()
	tmpDir, err := os.MkdirTemp(systemTmp, "amux-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repo with explicit settings
	cmd := exec.Command("git", "init", "--initial-branch=main")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		// Fallback for older git versions
		cmd = exec.Command("git", "init")
		cmd.Dir = tmpDir
		output, err := cmd.CombinedOutput()
		if err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("Failed to init git repo: %v, output: %s", err, output)
		}
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to configure git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to configure git name: %v", err)
	}

	// Set git to not use templates (which might interfere)
	cmd = exec.Command("git", "config", "init.templateDir", "")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	// Create initial commit
	readmePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repository\n"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create README: %v", err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to add README: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Create main branch
	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = tmpDir
	_ = cmd.Run() // Ignore error - might already be on main

	return tmpDir
}
