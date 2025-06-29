package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_LoadTasks(t *testing.T) {
	tests := []struct {
		name    string
		tasks   []*Task
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid tasks",
			tasks: []*Task{
				{
					Name:    "build",
					Command: "go build",
				},
				{
					Name:    "test",
					Command: "go test",
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
			errMsg:  "duplicate task name: build",
		},
		{
			name: "invalid task",
			tasks: []*Task{
				{
					Name: "invalid",
					// Missing command
				},
			},
			wantErr: true,
			errMsg:  "invalid task invalid",
		},
		{
			name: "non-existent dependency",
			tasks: []*Task{
				{
					Name:      "build",
					Command:   "go build",
					DependsOn: []string{"prepare"},
				},
			},
			wantErr: true,
			errMsg:  "depends on non-existent task: prepare",
		},
		{
			name: "valid dependency",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager()
			err := m.LoadTasks(tt.tasks)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Len(t, m.tasks, len(tt.tasks))
			}
		})
	}
}

func TestManager_GetTask(t *testing.T) {
	m := NewManager()
	tasks := []*Task{
		{
			Name:    "build",
			Command: "go build",
		},
		{
			Name:    "test",
			Command: "go test",
		},
	}
	require.NoError(t, m.LoadTasks(tasks))

	tests := []struct {
		name     string
		taskName string
		wantErr  bool
		want     *Task
	}{
		{
			name:     "existing task",
			taskName: "build",
			wantErr:  false,
			want:     tasks[0],
		},
		{
			name:     "non-existent task",
			taskName: "deploy",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := m.GetTask(tt.taskName)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.Name, task.Name)
				assert.Equal(t, tt.want.Command, task.Command)
			}
		})
	}
}

func TestManager_ListTasks(t *testing.T) {
	m := NewManager()

	// Empty manager
	tasks := m.ListTasks()
	assert.Empty(t, tasks)

	// With tasks
	loadTasks := []*Task{
		{
			Name:    "build",
			Command: "go build",
		},
		{
			Name:    "test",
			Command: "go test",
		},
	}
	require.NoError(t, m.LoadTasks(loadTasks))

	tasks = m.ListTasks()
	assert.Len(t, tasks, 2)

	// Check that we got all tasks
	names := make(map[string]bool)
	for _, task := range tasks {
		names[task.Name] = true
	}
	assert.True(t, names["build"])
	assert.True(t, names["test"])
}

func TestManager_HasTask(t *testing.T) {
	m := NewManager()
	tasks := []*Task{
		{
			Name:    "build",
			Command: "go build",
		},
	}
	require.NoError(t, m.LoadTasks(tasks))

	assert.True(t, m.HasTask("build"))
	assert.False(t, m.HasTask("test"))
}

func TestManager_GetTaskNames(t *testing.T) {
	m := NewManager()

	// Empty manager
	names := m.GetTaskNames()
	assert.Empty(t, names)

	// With tasks
	tasks := []*Task{
		{
			Name:    "build",
			Command: "go build",
		},
		{
			Name:    "test",
			Command: "go test",
		},
		{
			Name:    "lint",
			Command: "golangci-lint run",
		},
	}
	require.NoError(t, m.LoadTasks(tasks))

	names = m.GetTaskNames()
	assert.Len(t, names, 3)

	// Check that we got all names
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}
	assert.True(t, nameMap["build"])
	assert.True(t, nameMap["test"])
	assert.True(t, nameMap["lint"])
}
