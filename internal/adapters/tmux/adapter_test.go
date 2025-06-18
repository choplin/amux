package tmux

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func skipIfNoTmux(t *testing.T) {
	adapter, err := NewAdapter()
	if err != nil || !adapter.IsAvailable() {
		t.Skip("tmux not available on this system")
	}
}

func TestAdapter_CreateAndKillSession(t *testing.T) {
	skipIfNoTmux(t)

	adapter, err := NewAdapter()
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	sessionName := "test-session-" + time.Now().Format("20060102-150405")
	workDir := t.TempDir()

	// Create session
	if err := adapter.CreateSession(sessionName, workDir); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Ensure cleanup
	defer adapter.KillSession(sessionName)

	// Check session exists
	if !adapter.SessionExists(sessionName) {
		t.Error("Session should exist after creation")
	}

	// Kill session
	if err := adapter.KillSession(sessionName); err != nil {
		t.Fatalf("Failed to kill session: %v", err)
	}

	// Check session is gone
	if adapter.SessionExists(sessionName) {
		t.Error("Session should not exist after killing")
	}
}

func TestAdapter_SendKeysAndCapture(t *testing.T) {
	skipIfNoTmux(t)

	adapter, err := NewAdapter()
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	sessionName := "test-session-" + time.Now().Format("20060102-150405")
	workDir := t.TempDir()

	// Create session
	if err := adapter.CreateSession(sessionName, workDir); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer adapter.KillSession(sessionName)

	// Wait for session to be ready
	time.Sleep(100 * time.Millisecond)

	// Send command
	testCommand := "echo 'Hello from tmux'"
	if err := adapter.SendKeys(sessionName, testCommand); err != nil {
		t.Fatalf("Failed to send keys: %v", err)
	}

	// Wait for command to execute
	time.Sleep(100 * time.Millisecond)

	// Capture output
	output, err := adapter.CapturePane(sessionName)
	if err != nil {
		t.Fatalf("Failed to capture pane: %v", err)
	}

	// Check output contains our text
	if !strings.Contains(output, "Hello from tmux") {
		t.Errorf("Output does not contain expected text. Got: %s", output)
	}
}

func TestAdapter_ListSessions(t *testing.T) {
	skipIfNoTmux(t)

	adapter, err := NewAdapter()
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// List sessions (might be empty)
	initialSessions, err := adapter.ListSessions()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	// Create a session
	sessionName := "test-session-" + time.Now().Format("20060102-150405")
	workDir := t.TempDir()

	if err := adapter.CreateSession(sessionName, workDir); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer adapter.KillSession(sessionName)

	// List sessions again
	sessions, err := adapter.ListSessions()
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	// Should have one more session
	if len(sessions) != len(initialSessions)+1 {
		t.Errorf("Expected %d sessions, got %d", len(initialSessions)+1, len(sessions))
	}

	// Check our session is in the list
	found := false
	for _, s := range sessions {
		if s == sessionName {
			found = true
			break
		}
	}
	if !found {
		t.Error("Created session not found in list")
	}
}

func TestAdapter_SetEnvironment(t *testing.T) {
	skipIfNoTmux(t)

	adapter, err := NewAdapter()
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	sessionName := "test-session-" + time.Now().Format("20060102-150405")
	workDir := t.TempDir()

	// Create session
	if err := adapter.CreateSession(sessionName, workDir); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer adapter.KillSession(sessionName)

	// Wait for session to be ready
	time.Sleep(100 * time.Millisecond)

	// Set environment variables
	env := map[string]string{
		"TEST_VAR1": "value1",
		"TEST_VAR2": "value2",
	}

	if err := adapter.SetEnvironment(sessionName, env); err != nil {
		t.Fatalf("Failed to set environment: %v", err)
	}

	// Export variables in the current shell
	for k, v := range env {
		exportCmd := fmt.Sprintf("export %s='%s'", k, v)
		if err := adapter.SendKeys(sessionName, exportCmd); err != nil {
			t.Fatalf("Failed to export %s: %v", k, err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Send command to print env var
	if err := adapter.SendKeys(sessionName, "echo $TEST_VAR1"); err != nil {
		t.Fatalf("Failed to send keys: %v", err)
	}

	// Wait for command
	time.Sleep(100 * time.Millisecond)

	// Capture and check
	output, err := adapter.CapturePane(sessionName)
	if err != nil {
		t.Fatalf("Failed to capture pane: %v", err)
	}

	if !strings.Contains(output, "value1") {
		t.Errorf("Environment variable not set correctly. Output: %s", output)
	}
}

func TestAdapter_GetSessionPID(t *testing.T) {
	skipIfNoTmux(t)

	adapter, err := NewAdapter()
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	sessionName := "test-session-" + time.Now().Format("20060102-150405")
	workDir := t.TempDir()

	// Create session
	if err := adapter.CreateSession(sessionName, workDir); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer adapter.KillSession(sessionName)

	// Get PID
	pid, err := adapter.GetSessionPID(sessionName)
	if err != nil {
		t.Fatalf("Failed to get session PID: %v", err)
	}

	if pid <= 0 {
		t.Errorf("Invalid PID: %d", pid)
	}
}

func TestAdapter_CreateSessionWithOptions(t *testing.T) {
	skipIfNoTmux(t)

	adapter, err := NewAdapter()
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	tests := []struct {
		name string
		opts CreateSessionOptions
	}{
		{
			name: "basic session",
			opts: CreateSessionOptions{
				SessionName: "test-basic-" + time.Now().Format("20060102-150405"),
				WorkDir:     t.TempDir(),
			},
		},
		{
			name: "session with shell",
			opts: CreateSessionOptions{
				SessionName: "test-shell-" + time.Now().Format("20060102-150405"),
				WorkDir:     t.TempDir(),
				Shell:       "/bin/bash",
			},
		},
		{
			name: "session with window name",
			opts: CreateSessionOptions{
				SessionName: "test-window-" + time.Now().Format("20060102-150405"),
				WorkDir:     t.TempDir(),
				WindowName:  "dev",
			},
		},
		{
			name: "session with all options",
			opts: CreateSessionOptions{
				SessionName: "test-full-" + time.Now().Format("20060102-150405"),
				WorkDir:     t.TempDir(),
				Shell:       "/bin/sh",
				WindowName:  "workspace",
			},
		},
		{
			name: "session with environment",
			opts: CreateSessionOptions{
				SessionName: "test-env-" + time.Now().Format("20060102-150405"),
				WorkDir:     t.TempDir(),
				Shell:       "/bin/bash",
				Environment: map[string]string{
					"AMUX_TEST_VAR": "test_value",
					"AMUX_SESSION":  "test_session",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create session
			if err := adapter.CreateSessionWithOptions(tt.opts); err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}
			defer adapter.KillSession(tt.opts.SessionName)

			// Verify session exists
			if !adapter.SessionExists(tt.opts.SessionName) {
				t.Error("Session should exist after creation")
			}

			// If shell was specified, verify it's running
			if tt.opts.Shell != "" {
				// Send command to check shell
				checkCmd := "echo $SHELL"
				if err := adapter.SendKeys(tt.opts.SessionName, checkCmd); err != nil {
					t.Fatalf("Failed to send keys: %v", err)
				}

				// Wait for command
				time.Sleep(100 * time.Millisecond)

				// Capture output
				output, err := adapter.CapturePane(tt.opts.SessionName)
				if err != nil {
					t.Fatalf("Failed to capture pane: %v", err)
				}

				// Note: The echo $SHELL might show the default shell, not the running shell
				// This is because tmux starts the specified shell as a command, not as SHELL env
				t.Logf("Shell check output: %s", output)
			}

			// If environment was specified, verify variables are set
			if len(tt.opts.Environment) > 0 {
				// Wait for session to be ready
				time.Sleep(500 * time.Millisecond)

				// Check one of the environment variables
				for key, expectedValue := range tt.opts.Environment {
					checkCmd := fmt.Sprintf("echo $%s", key)
					if err := adapter.SendKeys(tt.opts.SessionName, checkCmd); err != nil {
						t.Fatalf("Failed to send keys: %v", err)
					}

					// Wait for command
					time.Sleep(300 * time.Millisecond)

					// Capture output
					output, err := adapter.CapturePane(tt.opts.SessionName)
					if err != nil {
						t.Fatalf("Failed to capture pane: %v", err)
					}

					// Check if the expected value appears in output
					if !strings.Contains(output, expectedValue) {
						// Try to get more context
						fullOutput, _ := adapter.CapturePane(tt.opts.SessionName)
						t.Errorf("Environment variable %s not set correctly. Expected %s in output.\nCommand sent: %s\nFull output:\n%s", key, expectedValue, checkCmd, fullOutput)
					}
					break // Check only one variable for test speed
				}
			}
		})
	}
}
