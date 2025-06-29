package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_Validate(t *testing.T) {
	tests := []struct {
		name    string
		task    Task
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid oneshot task",
			task: Task{
				Name:      "test",
				Command:   "echo hello",
				Lifecycle: LifecycleOneshot,
			},
			wantErr: false,
		},
		{
			name: "valid daemon task",
			task: Task{
				Name:      "server",
				Command:   "npm run dev",
				Lifecycle: LifecycleDaemon,
			},
			wantErr: false,
		},
		{
			name: "default lifecycle",
			task: Task{
				Name:    "test",
				Command: "echo hello",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			task: Task{
				Command: "echo hello",
			},
			wantErr: true,
			errMsg:  "task name cannot be empty",
		},
		{
			name: "empty command",
			task: Task{
				Name: "test",
			},
			wantErr: true,
			errMsg:  "task command cannot be empty",
		},
		{
			name: "invalid lifecycle",
			task: Task{
				Name:      "test",
				Command:   "echo hello",
				Lifecycle: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid lifecycle type",
		},
		{
			name: "timeout on daemon task",
			task: Task{
				Name:      "server",
				Command:   "npm run dev",
				Lifecycle: LifecycleDaemon,
				Timeout:   "30s",
			},
			wantErr: true,
			errMsg:  "timeout can only be specified for oneshot tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTask_IsOneshot(t *testing.T) {
	tests := []struct {
		name     string
		task     Task
		expected bool
	}{
		{
			name: "explicit oneshot",
			task: Task{
				Lifecycle: LifecycleOneshot,
			},
			expected: true,
		},
		{
			name: "explicit daemon",
			task: Task{
				Lifecycle: LifecycleDaemon,
			},
			expected: false,
		},
		{
			name:     "default",
			task:     Task{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.task.IsOneshot())
		})
	}
}

func TestTask_IsDaemon(t *testing.T) {
	tests := []struct {
		name     string
		task     Task
		expected bool
	}{
		{
			name: "explicit daemon",
			task: Task{
				Lifecycle: LifecycleDaemon,
			},
			expected: true,
		},
		{
			name: "explicit oneshot",
			task: Task{
				Lifecycle: LifecycleOneshot,
			},
			expected: false,
		},
		{
			name:     "default",
			task:     Task{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.task.IsDaemon())
		})
	}
}

func TestValidateLifecycleType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid oneshot",
			input:   "oneshot",
			wantErr: false,
		},
		{
			name:    "valid daemon",
			input:   "daemon",
			wantErr: false,
		},
		{
			name:    "case insensitive",
			input:   "ONESHOT",
			wantErr: false,
		},
		{
			name:    "invalid",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLifecycleType(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
