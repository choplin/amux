package task

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_ExecuteTask(t *testing.T) {
	// Create test manager and executor
	manager := NewManager()
	executor := NewExecutor(manager)

	// Use OS-appropriate temp directory
	tempDir := t.TempDir()

	// Load test tasks
	tasks := []*Task{
		{
			Name:    "echo",
			Command: "echo 'Hello, World!'",
		},
		{
			Name:      "pwd",
			Command:   "pwd",
			Lifecycle: LifecycleOneshot,
		},
		{
			Name:      "timeout-test",
			Command:   "sleep 5",
			Lifecycle: LifecycleOneshot,
			Timeout:   "100ms",
		},
		{
			Name:    "env-test",
			Command: "echo $TEST_VAR",
			Env: map[string]string{
				"TEST_VAR": "test-value",
			},
		},
	}
	require.NoError(t, manager.LoadTasks(tasks))

	tests := []struct {
		name       string
		taskName   string
		workingDir string
		wantErr    bool
		errMsg     string
		timeout    time.Duration
	}{
		{
			name:       "execute simple echo task",
			taskName:   "echo",
			workingDir: tempDir,
			wantErr:    false,
		},
		{
			name:       "execute pwd task",
			taskName:   "pwd",
			workingDir: tempDir,
			wantErr:    false,
		},
		{
			name:       "task with timeout",
			taskName:   "timeout-test",
			workingDir: tempDir,
			wantErr:    true,
			errMsg:     "task timed out after",
			timeout:    200 * time.Millisecond,
		},
		{
			name:       "non-existent task",
			taskName:   "non-existent",
			workingDir: tempDir,
			wantErr:    true,
			errMsg:     "task not found",
		},
		{
			name:       "task with environment variable",
			taskName:   "env-test",
			workingDir: tempDir,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.timeout)
				defer cancel()
			}

			err := executor.ExecuteTask(ctx, tt.taskName, tt.workingDir)
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

func TestExecutor_ValidateDependencies(t *testing.T) {
	manager := NewManager()
	executor := NewExecutor(manager)

	// Load test tasks with dependencies
	tasks := []*Task{
		{
			Name:    "prepare",
			Command: "echo preparing",
		},
		{
			Name:      "build",
			Command:   "echo building",
			DependsOn: []string{"prepare"},
		},
		{
			Name:      "test",
			Command:   "echo testing",
			DependsOn: []string{"build"},
		},
		{
			Name:      "deploy",
			Command:   "echo deploying",
			DependsOn: []string{"build", "test"},
		},
	}
	require.NoError(t, manager.LoadTasks(tasks))

	tests := []struct {
		name     string
		taskName string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "task with no dependencies",
			taskName: "prepare",
			wantErr:  false,
		},
		{
			name:     "task with single dependency",
			taskName: "build",
			wantErr:  false,
		},
		{
			name:     "task with transitive dependencies",
			taskName: "test",
			wantErr:  false,
		},
		{
			name:     "task with multiple dependencies",
			taskName: "deploy",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateDependencies(tt.taskName)
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

func TestExecutor_DaemonTask(t *testing.T) {
	// Note: This is a basic test for daemon task execution
	// Full daemon management will be implemented in Phase 3
	manager := NewManager()
	executor := NewExecutor(manager)

	// Use OS-appropriate temp directory
	tempDir := t.TempDir()

	tasks := []*Task{
		{
			Name:      "daemon-test",
			Command:   "sleep 0.1",
			Lifecycle: LifecycleDaemon,
		},
	}
	require.NoError(t, manager.LoadTasks(tasks))

	ctx := context.Background()
	err := executor.ExecuteTask(ctx, "daemon-test", tempDir)
	require.NoError(t, err)
	// The daemon task should start successfully
	// In Phase 3, we'll add proper daemon management
}
