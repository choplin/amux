package hooks_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aki/amux/internal/core/hooks"
)

func TestExecutor_ExecuteHooks(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("executes command successfully", func(t *testing.T) {
		hook := hooks.Hook{
			Name:    "Test command",
			Command: "echo hello",
			Timeout: "1s",
			OnError: hooks.ErrorStrategyFail,
		}

		var output bytes.Buffer
		executor := hooks.NewExecutor(tmpDir, nil).WithOutput(&output)

		err := executor.ExecuteHooks(context.Background(), hooks.EventWorkspaceCreate, []hooks.Hook{hook})
		require.NoError(t, err)

		assert.Contains(t, output.String(), "hello")
	})

	t.Run("executes script successfully", func(t *testing.T) {
		// Create a test script
		scriptPath := filepath.Join(tmpDir, "test.sh")
		scriptContent := "#!/bin/sh\necho script output"
		err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755)
		require.NoError(t, err)

		hook := hooks.Hook{
			Name:    "Test script",
			Script:  scriptPath,
			Timeout: "5s",
			OnError: hooks.ErrorStrategyFail,
		}

		var output bytes.Buffer
		executor := hooks.NewExecutor(tmpDir, nil).WithOutput(&output)

		err = executor.ExecuteHooks(context.Background(), hooks.EventWorkspaceCreate, []hooks.Hook{hook})
		if runtime.GOOS == "windows" {
			// Scripts might not work on Windows in CI
			t.Skip("Skipping script test on Windows")
		}
		require.NoError(t, err)

		assert.Contains(t, output.String(), "script output")
	})

	t.Run("handles command failure with fail strategy", func(t *testing.T) {
		// Use a command that exists cross-platform
		cmd := "false"
		if runtime.GOOS == "windows" {
			cmd = "cmd /c exit 1"
		}

		hook := hooks.Hook{
			Name:    "Failing command",
			Command: cmd,
			Timeout: "1s",
			OnError: hooks.ErrorStrategyFail,
		}

		executor := hooks.NewExecutor(tmpDir, nil).WithOutput(&bytes.Buffer{})

		err := executor.ExecuteHooks(context.Background(), hooks.EventWorkspaceCreate, []hooks.Hook{hook})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed")
	})

	t.Run("continues on failure with warn strategy", func(t *testing.T) {
		// Use a command that exists cross-platform
		failCmd := "false"
		if runtime.GOOS == "windows" {
			failCmd = "cmd /c exit 1"
		}

		testHooks := []hooks.Hook{
			{
				Name:    "Failing command",
				Command: failCmd,
				Timeout: "1s",
				OnError: hooks.ErrorStrategyWarn,
			},
			{
				Name:    "Success command",
				Command: "echo success",
				Timeout: "1s",
				OnError: hooks.ErrorStrategyFail,
			},
		}

		var output bytes.Buffer
		executor := hooks.NewExecutor(tmpDir, nil).WithOutput(&output)

		err := executor.ExecuteHooks(context.Background(), hooks.EventWorkspaceCreate, testHooks)
		require.NoError(t, err)

		assert.Contains(t, output.String(), "success")
	})

	t.Run("uses environment variables", func(t *testing.T) {
		// Create a test script that uses env vars
		scriptPath := filepath.Join(tmpDir, "envtest.sh")
		scriptContent := "#!/bin/sh\necho \"Value: $TEST_VAR\""
		err := os.WriteFile(scriptPath, []byte(scriptContent), 0o755)
		require.NoError(t, err)

		hook := hooks.Hook{
			Name:    "Env test",
			Script:  scriptPath,
			Timeout: "5s", // Increase timeout to avoid flaky failures on first run
			OnError: hooks.ErrorStrategyFail,
			Env: map[string]string{
				"TEST_VAR": "test_value",
			},
		}

		var output bytes.Buffer
		executor := hooks.NewExecutor(tmpDir, nil).WithOutput(&output)

		err = executor.ExecuteHooks(context.Background(), hooks.EventWorkspaceCreate, []hooks.Hook{hook})
		if runtime.GOOS == "windows" {
			// Scripts might not work on Windows in CI
			t.Skip("Skipping env var test on Windows")
		}
		require.NoError(t, err)

		assert.Contains(t, output.String(), "test_value")
	})

	t.Run("dry run doesn't execute", func(t *testing.T) {
		hook := hooks.Hook{
			Name:    "Test command",
			Command: "echo should not appear",
			Timeout: "1s",
			OnError: hooks.ErrorStrategyFail,
		}

		var output bytes.Buffer
		executor := hooks.NewExecutor(tmpDir, nil).
			WithOutput(&output).
			WithDryRun(true)

		err := executor.ExecuteHooks(context.Background(), hooks.EventWorkspaceCreate, []hooks.Hook{hook})
		require.NoError(t, err)

		outputStr := output.String()
		assert.NotContains(t, outputStr, "should not appear")
		// The dry run message is printed to the terminal, not captured in output
		// Just verify the command wasn't actually executed
	})
}
