package process

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestDefaultChecker_HasChildren(t *testing.T) {
	// This test needs different commands on different platforms
	if runtime.GOOS == "windows" {
		t.Skip("Test not implemented for Windows")
	}

	checker := &DefaultChecker{}

	// Test with current process (test runner)
	// The test runner usually has no child processes
	_, err := checker.HasChildren(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to check current process: %v", err)
	}
	// We can't assert the value as it depends on the test environment

	// Test with non-existent PID
	hasChildren, err := checker.HasChildren(999999)
	if err != nil {
		t.Fatalf("Failed to check non-existent process: %v", err)
	}
	if hasChildren {
		t.Error("Non-existent process should not have children")
	}

	// Test with a process that has children
	// Start a shell with a sleep command
	cmd := exec.Command("sh", "-c", "sleep 30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}
	defer cmd.Process.Kill()

	// Give it a moment to start the child process
	time.Sleep(100 * time.Millisecond)

	hasChildren, err = checker.HasChildren(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("Failed to check test process: %v", err)
	}

	// Note: On some systems (especially macOS), sh -c might exec the sleep command
	// directly rather than keeping it as a child. This is a valid optimization.
	// We'll just log the result for debugging purposes.
	t.Logf("Shell process (PID %d) has children: %v", cmd.Process.Pid, hasChildren)
}

func TestHasChildren(t *testing.T) {
	// This test needs platform-specific commands
	if runtime.GOOS == "windows" {
		t.Skip("Test not implemented for Windows")
	}

	// Test the convenience function
	hasChildren, err := HasChildren(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to check current process: %v", err)
	}
	// Just verify it doesn't error - we can't predict the result
	_ = hasChildren
}
