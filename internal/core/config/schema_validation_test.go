package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			yaml: `version: "1.0"
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
			wantErr: false,
		},
		{
			name: "missing required version",
			yaml: `agents:
  claude:
    name: Claude
    type: tmux
    params:
      command: claude`,
			wantErr: true,
			errMsg:  "missing properties: 'version'",
		},
		{
			name:    "missing required agents",
			yaml:    `version: "1.0"`,
			wantErr: true,
			errMsg:  "missing properties: 'agents'",
		},
		{
			name: "invalid version",
			yaml: `version: "2.0"
agents:
  claude:
    name: Claude
    type: tmux
    params:
      command: claude`,
			wantErr: true,
			errMsg:  "value must be \"1.0\"",
		},
		{
			name: "missing agent name",
			yaml: `version: "1.0"
agents:
  claude:
    type: tmux
    params:
      command: claude`,
			wantErr: true,
			errMsg:  "missing properties: 'name'",
		},
		{
			name: "missing agent type",
			yaml: `version: "1.0"
agents:
  claude:
    name: Claude
    params:
      command: claude`,
			wantErr: true,
			errMsg:  "missing properties: 'type'",
		},
		{
			name: "invalid agent type",
			yaml: `version: "1.0"
agents:
  claude:
    name: Claude
    type: docker
    params:
      command: claude`,
			wantErr: true,
			errMsg:  "value must be \"tmux\"",
		},
		{
			name: "tmux agent missing params config",
			yaml: `version: "1.0"
agents:
  claude:
    name: Claude
    type: tmux`,
			wantErr: true,
			errMsg:  "missing properties: 'params'",
		},
		{
			name: "tmux config missing command",
			yaml: `version: "1.0"
agents:
  claude:
    name: Claude
    type: tmux
    params:
      shell: /bin/bash`,
			wantErr: true,
			errMsg:  "missing properties: 'command'",
		},
		{
			name: "additional properties not allowed",
			yaml: `version: "1.0"
agents:
  claude:
    name: Claude
    type: tmux
    unknown: value
    params:
      command: claude`,
			wantErr: true,
			errMsg:  "additionalProperties",
		},
		{
			name: "invalid agent id pattern",
			yaml: `version: "1.0"
agents:
  "invalid-@-id":
    name: Claude
    type: tmux
    params:
      command: claude`,
			wantErr: true,
			errMsg:  "additionalProperties 'invalid-@-id' not allowed",
		},
		{
			name: "invalid transport type",
			yaml: `version: "1.0"
mcp:
  transport:
    type: websocket
agents:
  claude:
    name: Claude
    type: tmux
    params:
      command: claude`,
			wantErr: true,
			errMsg:  "value must be one of \"stdio\", \"http\"",
		},
		{
			name: "valid with all optional fields",
			yaml: `version: "1.0"
mcp:
  transport:
    type: http
    http:
      port: 8080
      auth:
        type: bearer
        bearer: secret-token
agents:
  claude:
    name: Claude
    type: tmux
    description: Claude AI assistant
    environment:
      API_KEY: ${CLAUDE_API_KEY}
    workingDir: /tmp/claude
    tags:
      - ai
      - assistant
    params:
      command: claude
      shell: /bin/zsh
      windowName: claude-window
      detached: true`,
			wantErr: false,
		},
		{
			name: "valid tmux config with autoAttach",
			yaml: `version: "1.0"
agents:
  claude-interactive:
    name: Claude Interactive
    type: tmux
    params:
      command: claude
      autoAttach: true`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateYAML([]byte(tt.yaml))
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadWithValidation(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	t.Run("valid configuration", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "valid.yaml")
		validConfig := `version: "1.0"
agents:
  claude:
    name: Claude
    type: tmux
    params:
      command: claude`

		require.NoError(t, os.WriteFile(configPath, []byte(validConfig), 0o644))

		cfg, err := LoadWithValidation(configPath)
		require.NoError(t, err)
		assert.Equal(t, "1.0", cfg.Version)
		assert.Equal(t, "Claude", cfg.Agents["claude"].Name)
		assert.Equal(t, AgentTypeTmux, cfg.Agents["claude"].Type)

		// Verify params were properly unmarshaled
		agent := cfg.Agents["claude"]
		params, err := agent.GetTmuxParams()
		require.NoError(t, err)
		assert.Equal(t, "claude", params.Command)
	})

	t.Run("invalid configuration", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "invalid.yaml")
		invalidConfig := `version: "1.0"
agents:
  claude:
    name: Claude
    # missing type field`

		require.NoError(t, os.WriteFile(configPath, []byte(invalidConfig), 0o644))

		cfg, err := LoadWithValidation(configPath)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "invalid configuration")
	})

	t.Run("file not found", func(t *testing.T) {
		cfg, err := LoadWithValidation(filepath.Join(tmpDir, "nonexistent.yaml"))
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "configuration file not found")
	})

	t.Run("malformed YAML", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "malformed.yaml")
		malformedYAML := `version: "1.0"
  [invalid yaml`

		require.NoError(t, os.WriteFile(configPath, []byte(malformedYAML), 0o644))

		cfg, err := LoadWithValidation(configPath)
		assert.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to parse YAML")
	})
}

func TestValidateFile(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	t.Run("valid file", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "valid.yaml")
		validConfig := `version: "1.0"
agents:
  claude:
    name: Claude
    type: tmux
    params:
      command: claude`

		require.NoError(t, os.WriteFile(configPath, []byte(validConfig), 0o644))

		err := ValidateFile(configPath)
		assert.NoError(t, err)
	})

	t.Run("invalid file", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "invalid.yaml")
		invalidConfig := `version: "2.0"
agents: {}`

		require.NoError(t, os.WriteFile(configPath, []byte(invalidConfig), 0o644))

		err := ValidateFile(configPath)
		assert.Error(t, err)
	})

	t.Run("file not found", func(t *testing.T) {
		err := ValidateFile(filepath.Join(tmpDir, "nonexistent.yaml"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "configuration file not found")
	})
}

func TestSchemaValidation_TypeSpecificFields(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name: "invalid type not in enum",
			yaml: `version: "1.0"
agents:
  future-agent:
    name: Future Agent
    type: claude-code
    params:
      command: some-command`,
			wantErr: "value must be \"tmux\"",
		},
		{
			name: "tmux agent with unexpected field",
			yaml: `version: "1.0"
agents:
  tmux-agent:
    name: Tmux Agent
    type: tmux
    params:
      command: bash
    claudeCode:
      cliPath: /usr/local/bin/claude`,
			wantErr: "additionalProperties",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateYAML([]byte(tt.yaml))
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
