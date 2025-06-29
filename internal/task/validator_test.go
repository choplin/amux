package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidator_ValidateTask(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		task    *Task
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid task",
			task: &Task{
				Name:    "build",
				Command: "go build",
			},
			wantErr: false,
		},
		{
			name: "task with spaces in name",
			task: &Task{
				Name:    "  build  ",
				Command: "go build",
			},
			wantErr: true,
			errMsg:  "cannot have leading or trailing whitespace",
		},
		{
			name: "invalid timeout format",
			task: &Task{
				Name:      "build",
				Command:   "go build",
				Lifecycle: LifecycleOneshot,
				Timeout:   "invalid",
			},
			wantErr: true,
			errMsg:  "invalid timeout format",
		},
		{
			name: "valid timeout",
			task: &Task{
				Name:      "build",
				Command:   "go build",
				Lifecycle: LifecycleOneshot,
				Timeout:   "30s",
			},
			wantErr: false,
		},
		{
			name: "empty env key",
			task: &Task{
				Name:    "build",
				Command: "go build",
				Env: map[string]string{
					"": "value",
				},
			},
			wantErr: true,
			errMsg:  "environment variable name cannot be empty",
		},
		{
			name: "env key with equals",
			task: &Task{
				Name:    "build",
				Command: "go build",
				Env: map[string]string{
					"KEY=VALUE": "value",
				},
			},
			wantErr: true,
			errMsg:  "environment variable name cannot contain '='",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateTask(tt.task)
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

func TestValidator_ValidateTaskList(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		tasks   []*Task
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid task list",
			tasks: []*Task{
				{
					Name:    "prepare",
					Command: "go mod download",
				},
				{
					Name:      "build",
					Command:   "go build",
					DependsOn: []string{"prepare"},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate task names",
			tasks: []*Task{
				{
					Name:    "build",
					Command: "go build",
				},
				{
					Name:    "build",
					Command: "make build",
				},
			},
			wantErr: true,
			errMsg:  "duplicate task name",
		},
		{
			name: "missing dependency",
			tasks: []*Task{
				{
					Name:      "build",
					Command:   "go build",
					DependsOn: []string{"prepare"},
				},
			},
			wantErr: true,
			errMsg:  "depends on non-existent task",
		},
		{
			name: "circular dependency",
			tasks: []*Task{
				{
					Name:      "a",
					Command:   "echo a",
					DependsOn: []string{"b"},
				},
				{
					Name:      "b",
					Command:   "echo b",
					DependsOn: []string{"a"},
				},
			},
			wantErr: true,
			errMsg:  "circular dependency detected",
		},
		{
			name: "self dependency",
			tasks: []*Task{
				{
					Name:      "build",
					Command:   "go build",
					DependsOn: []string{"build"},
				},
			},
			wantErr: true,
			errMsg:  "circular dependency detected",
		},
		{
			name: "complex valid dependencies",
			tasks: []*Task{
				{
					Name:    "download",
					Command: "go mod download",
				},
				{
					Name:      "generate",
					Command:   "go generate",
					DependsOn: []string{"download"},
				},
				{
					Name:      "build",
					Command:   "go build",
					DependsOn: []string{"download", "generate"},
				},
				{
					Name:      "test",
					Command:   "go test",
					DependsOn: []string{"build"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateTaskList(tt.tasks)
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
