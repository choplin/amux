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
				Command: "claude",
			},
		},
		{
			name: "with autoAttach true",
			yaml: `command: claude
autoAttach: true`,
			expected: TmuxParams{
				Command:    "claude",
				AutoAttach: true,
			},
		},
		{
			name: "with autoAttach false",
			yaml: `command: claude
autoAttach: false`,
			expected: TmuxParams{
				Command:    "claude",
				AutoAttach: false,
			},
		},
		{
			name: "full params",
			yaml: `command: claude
shell: /bin/zsh
windowName: claude-window
detached: true
autoAttach: true`,
			expected: TmuxParams{
				Command:    "claude",
				Shell:      "/bin/zsh",
				WindowName: "claude-window",
				Detached:   true,
				AutoAttach: true,
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
	assert.Equal(t, "claude", tmuxParams.Command)
	assert.True(t, tmuxParams.AutoAttach)
}
