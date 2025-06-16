package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "config is nil",
		},
		{
			name: "valid config with tmux agent",
			config: &Config{
				Version: "1.0",
				Agents: map[string]Agent{
					"claude": {
						Name: "Claude",
						Type: "tmux",
						Tmux: &TmuxConfig{
							Command: "claude",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid agent - missing name",
			config: &Config{
				Version: "1.0",
				Agents: map[string]Agent{
					"claude": {
						Type: "tmux",
						Tmux: &TmuxConfig{
							Command: "claude",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid agent 'claude': name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
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

func TestValidateAgent(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		agent   *Agent
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil agent",
			id:      "test",
			agent:   nil,
			wantErr: true,
			errMsg:  "agent is nil",
		},
		{
			name: "missing name",
			id:   "test",
			agent: &Agent{
				Type: "tmux",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "missing type",
			id:   "test",
			agent: &Agent{
				Name: "Test Agent",
			},
			wantErr: true,
			errMsg:  "type is required",
		},
		{
			name: "unsupported type",
			id:   "test",
			agent: &Agent{
				Name: "Test Agent",
				Type: "unknown",
			},
			wantErr: true,
			errMsg:  "unsupported agent type: unknown",
		},
		{
			name: "tmux without config",
			id:   "test",
			agent: &Agent{
				Name: "Test Agent",
				Type: "tmux",
			},
			wantErr: true,
			errMsg:  "tmux configuration is required for type 'tmux'",
		},
		{
			name: "tmux without command",
			id:   "test",
			agent: &Agent{
				Name: "Test Agent",
				Type: "tmux",
				Tmux: &TmuxConfig{},
			},
			wantErr: true,
			errMsg:  "invalid tmux configuration: command is required",
		},
		{
			name: "valid tmux agent",
			id:   "test",
			agent: &Agent{
				Name:        "Test Agent",
				Type:        "tmux",
				Description: "A test agent",
				Environment: map[string]string{
					"KEY": "value",
				},
				Tmux: &TmuxConfig{
					Command:    "test-command",
					Shell:      "/bin/bash",
					WindowName: "test-window",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgent(tt.id, tt.agent)
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

func TestValidateTmuxConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *TmuxConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "tmux config is nil",
		},
		{
			name: "missing command",
			config: &TmuxConfig{
				Shell: "/bin/bash",
			},
			wantErr: true,
			errMsg:  "command is required",
		},
		{
			name: "valid config",
			config: &TmuxConfig{
				Command:    "claude",
				Shell:      "/bin/bash",
				WindowName: "claude-session",
				Detached:   true,
			},
			wantErr: false,
		},
		{
			name: "minimal valid config",
			config: &TmuxConfig{
				Command: "claude",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTmuxConfig(tt.config)
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
