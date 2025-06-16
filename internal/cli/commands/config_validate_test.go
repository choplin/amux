package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidateCommand(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	amuxDir := filepath.Join(tmpDir, ".amux")
	require.NoError(t, os.MkdirAll(amuxDir, 0o755))

	// Change to test directory
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(oldWd)

	tests := []struct {
		name           string
		config         string
		expectedError  bool
		expectedOutput string
	}{
		{
			name: "valid configuration",
			config: `version: "1.0"
project:
  name: test-project
  repository: https://github.com/test/project.git
  defaultAgent: claude
mcp:
  transport:
    type: stdio
agents:
  claude:
    name: Claude
    type: tmux
    description: Test agent
    params:
      command: claude`,
			expectedError:  false,
			expectedOutput: "Configuration is valid",
		},
		{
			name: "missing agent type",
			config: `version: "1.0"
project:
  name: test-project
agents:
  claude:
    name: Claude
    params:
      command: claude`,
			expectedError:  true,
			expectedOutput: "Configuration validation failed",
		},
		{
			name: "missing agent name",
			config: `version: "1.0"
project:
  name: test-project
agents:
  claude:
    type: tmux
    params:
      command: claude`,
			expectedError:  true,
			expectedOutput: "Configuration validation failed",
		},
		{
			name: "unsupported agent type",
			config: `version: "1.0"
project:
  name: test-project
agents:
  claude:
    name: Claude
    type: unsupported
    params:
      command: claude`,
			expectedError:  true,
			expectedOutput: "Configuration validation failed",
		},
		{
			name: "missing params command",
			config: `version: "1.0"
project:
  name: test-project
agents:
  claude:
    name: Claude
    type: tmux
    params:
      shell: /bin/bash`,
			expectedError:  true,
			expectedOutput: "Configuration validation failed",
		},
		{
			name: "missing params config for tmux type",
			config: `version: "1.0"
project:
  name: test-project
agents:
  claude:
    name: Claude
    type: tmux`,
			expectedError:  true,
			expectedOutput: "Configuration validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write config file
			configPath := filepath.Join(amuxDir, "config.yaml")
			require.NoError(t, os.WriteFile(configPath, []byte(tt.config), 0o644))

			// Execute command
			cmd := configValidateCmd()

			// Capture stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			err := cmd.Execute()

			// Restore stdout and stderr
			w.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr

			// Read captured output
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Contains(t, output, tt.expectedOutput)
		})
	}
}

func TestConfigValidateCommandVerbose(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	amuxDir := filepath.Join(tmpDir, ".amux")
	require.NoError(t, os.MkdirAll(amuxDir, 0o755))

	// Change to test directory
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(oldWd)

	// Write valid config
	validConfig := `version: "1.0"
project:
  name: test-project
  repository: https://github.com/test/project.git
  defaultAgent: claude
mcp:
  transport:
    type: stdio
agents:
  claude:
    name: Claude
    type: tmux
    description: Test agent
    environment:
      TEST_VAR: test_value
    params:
      command: claude
      shell: /bin/bash
      windowName: claude-window
  aider:
    name: Aider
    type: tmux
    params:
      command: aider`

	configPath := filepath.Join(amuxDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(validConfig), 0o644))

	// Execute command with verbose flag
	cmd := configValidateCmd()
	cmd.SetArgs([]string{"--verbose"})

	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	err := cmd.Execute()
	assert.NoError(t, err)

	// Restore stdout and stderr
	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for verbose output
	assert.Contains(t, output, "Configuration is valid")
	assert.Contains(t, output, "Configuration details:")
	assert.Contains(t, output, "Version: 1.0")
	assert.Contains(t, output, "Project: test-project")
	assert.Contains(t, output, "Repository: https://github.com/test/project.git")
	assert.Contains(t, output, "Default Agent: claude")
	assert.Contains(t, output, "MCP Configuration:")
	assert.Contains(t, output, "Transport: stdio")
	assert.Contains(t, output, "Agents (2 configured):")
	assert.Contains(t, output, "claude:")
	assert.Contains(t, output, "Name: Claude")
	assert.Contains(t, output, "Type: tmux")
	assert.Contains(t, output, "Command: claude")
	assert.Contains(t, output, "Shell: /bin/bash")
	assert.Contains(t, output, "Window Name: claude-window")
	assert.Contains(t, output, "Description: Test agent")
	assert.Contains(t, output, "Environment Variables: 1")
	assert.Contains(t, output, "aider:")
	assert.Contains(t, output, "Name: Aider")
}

func TestConfigValidateCommandNotInProject(t *testing.T) {
	// Create a temporary directory without .amux
	tmpDir := t.TempDir()

	// Change to test directory
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(oldWd)

	// Execute command
	cmd := configValidateCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in an Amux project")
}
