package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestTmuxParamsUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected TmuxParams
	}{
		{
			name: "basic command only",
			yaml: `command: claude`,
			expected: TmuxParams{
				Command: Command{Single: "claude"},
			},
		},
		{
			name: "command as array",
			yaml: `command: ["python", "-m", "http.server", "8080"]`,
			expected: TmuxParams{
				Command: Command{Array: []string{"python", "-m", "http.server", "8080"}},
			},
		},
		{
			name: "with autoAttach true",
			yaml: `command: claude
autoAttach: true`,
			expected: TmuxParams{
				Command:    Command{Single: "claude"},
				AutoAttach: true,
			},
		},
		{
			name: "with autoAttach false",
			yaml: `command: claude
autoAttach: false`,
			expected: TmuxParams{
				Command:    Command{Single: "claude"},
				AutoAttach: false,
			},
		},
		{
			name: "full params",
			yaml: `command: claude
windowName: claude-window
detached: true
autoAttach: true`,
			expected: TmuxParams{
				Command:    Command{Single: "claude"},
				WindowName: "claude-window",
				Detached:   true,
				AutoAttach: true,
			},
		},
		{
			name: "shell command with full path",
			yaml: `command: /bin/zsh`,
			expected: TmuxParams{
				Command: Command{Single: "/bin/zsh"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params TmuxParams
			err := yaml.Unmarshal([]byte(tt.yaml), &params)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, params)
		})
	}
}

func TestAgentUnmarshalWithAutoAttach(t *testing.T) {
	yamlStr := `
name: Claude Interactive
type: tmux
description: Interactive Claude session
params:
  command: claude
  autoAttach: true
`

	var agent Agent
	err := yaml.Unmarshal([]byte(yamlStr), &agent)
	require.NoError(t, err)

	assert.Equal(t, "Claude Interactive", agent.Name)
	assert.Equal(t, AgentTypeTmux, agent.Type)
	assert.Equal(t, "Interactive Claude session", agent.Description)

	// Check TmuxParams
	tmuxParams, err := agent.GetTmuxParams()
	require.NoError(t, err)
	require.NotNil(t, tmuxParams)
	assert.Equal(t, "claude", tmuxParams.Command.Single)
	assert.False(t, tmuxParams.Command.IsArray())
	assert.True(t, tmuxParams.AutoAttach)
}

func TestCommandUnmarshalYAML(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expected    Command
		shouldError bool
	}{
		{
			name:     "string command",
			yaml:     `"echo hello"`,
			expected: Command{Single: "echo hello"},
		},
		{
			name:     "array command",
			yaml:     `["python", "-m", "http.server", "8080"]`,
			expected: Command{Array: []string{"python", "-m", "http.server", "8080"}},
		},
		{
			name:     "empty string",
			yaml:     `""`,
			expected: Command{Single: ""},
		},
		{
			name:     "empty array",
			yaml:     `[]`,
			expected: Command{Array: []string{}},
		},
		{
			name:        "invalid type",
			yaml:        `123`,
			shouldError: true,
		},
		{
			name:     "null value",
			yaml:     `null`,
			expected: Command{}, // null results in zero value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd Command
			err := yaml.Unmarshal([]byte(tt.yaml), &cmd)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, cmd)
			}
		})
	}
}

func TestCommandMethods(t *testing.T) {
	t.Run("IsArray", func(t *testing.T) {
		stringCmd := Command{Single: "echo hello"}
		assert.False(t, stringCmd.IsArray())

		arrayCmd := Command{Array: []string{"echo", "hello"}}
		assert.True(t, arrayCmd.IsArray())
	})

	t.Run("String", func(t *testing.T) {
		stringCmd := Command{Single: "echo hello"}
		assert.Equal(t, "echo hello", stringCmd.String())

		arrayCmd := Command{Array: []string{"echo", "hello"}}
		assert.Equal(t, "echo", arrayCmd.String())

		emptyArrayCmd := Command{Array: []string{}}
		assert.Equal(t, "", emptyArrayCmd.String())
	})
}
