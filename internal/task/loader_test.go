package task

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromYAML(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Test cases
	tests := []struct {
		name     string
		content  string
		filename string
		wantErr  bool
		validate func(t *testing.T, tasks []*Task)
	}{
		{
			name: "valid single task",
			content: `- name: build
  command: go build
  description: Build the application`,
			filename: "valid_single.yaml",
			wantErr:  false,
			validate: func(t *testing.T, tasks []*Task) {
				require.Len(t, tasks, 1)
				assert.Equal(t, "build", tasks[0].Name)
				assert.Equal(t, "go build", tasks[0].Command)
				assert.Equal(t, "Build the application", tasks[0].Description)
			},
		},
		{
			name: "multiple tasks",
			content: `- name: build
  command: go build
- name: test
  command: go test ./...
  lifecycle: oneshot
- name: server
  command: npm run dev
  lifecycle: daemon`,
			filename: "multiple.yaml",
			wantErr:  false,
			validate: func(t *testing.T, tasks []*Task) {
				require.Len(t, tasks, 3)
				assert.Equal(t, "build", tasks[0].Name)
				assert.Equal(t, "test", tasks[1].Name)
				assert.Equal(t, LifecycleOneshot, tasks[1].Lifecycle)
				assert.Equal(t, "server", tasks[2].Name)
				assert.Equal(t, LifecycleDaemon, tasks[2].Lifecycle)
			},
		},
		{
			name: "task with dependencies",
			content: `- name: prepare
  command: go mod download
- name: build
  command: go build
  depends_on:
    - prepare`,
			filename: "deps.yaml",
			wantErr:  false,
			validate: func(t *testing.T, tasks []*Task) {
				require.Len(t, tasks, 2)
				assert.Equal(t, "build", tasks[1].Name)
				assert.Equal(t, []string{"prepare"}, tasks[1].DependsOn)
			},
		},
		{
			name: "task with environment",
			content: `- name: test
  command: go test
  env:
    GOOS: linux
    GOARCH: amd64`,
			filename: "env.yaml",
			wantErr:  false,
			validate: func(t *testing.T, tasks []*Task) {
				require.Len(t, tasks, 1)
				assert.Equal(t, "linux", tasks[0].Env["GOOS"])
				assert.Equal(t, "amd64", tasks[0].Env["GOARCH"])
			},
		},
		{
			name:     "invalid yaml",
			content:  `invalid: yaml: content`,
			filename: "invalid.yaml",
			wantErr:  true,
		},
		{
			name:     "empty file",
			content:  "",
			filename: "empty.yaml",
			wantErr:  false,
			validate: func(t *testing.T, tasks []*Task) {
				assert.Empty(t, tasks)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tmpDir, tt.filename)
			err := os.WriteFile(filePath, []byte(tt.content), 0o644)
			require.NoError(t, err)

			// Load tasks
			tasks, err := LoadFromYAML(filePath)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, tasks)
				}
			}
		})
	}

	// Test non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		_, err := LoadFromYAML("/non/existent/file.yaml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read task file")
	})
}

func TestLoadFromDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "tasks")
	err := os.MkdirAll(subDir, 0o755)
	require.NoError(t, err)

	// Create test files
	files := map[string]string{
		"build.yaml": `- name: build
  command: go build`,
		"test.yml": `- name: test
  command: go test`,
		"tasks/dev.yaml": `- name: dev
  command: npm run dev`,
		"ignore.txt": `This should be ignored`,
		".hidden.yaml": `- name: hidden
  command: echo hidden`,
	}

	for filename, content := range files {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Test loading from directory
	tasks, err := LoadFromDirectory(tmpDir)
	require.NoError(t, err)

	// Check that we loaded the right tasks
	taskMap := make(map[string]*Task)
	for _, task := range tasks {
		taskMap[task.Name] = task
	}

	// Should have loaded build, test, dev, and hidden tasks
	assert.Len(t, taskMap, 4)
	assert.Contains(t, taskMap, "build")
	assert.Contains(t, taskMap, "test")
	assert.Contains(t, taskMap, "dev")
	assert.Contains(t, taskMap, "hidden")

	// Test non-existent directory
	tasks, err = LoadFromDirectory("/non/existent/directory")
	require.NoError(t, err)
	assert.Empty(t, tasks)

	// Test with file instead of directory
	filePath := filepath.Join(tmpDir, "notadir.txt")
	err = os.WriteFile(filePath, []byte("content"), 0o644)
	require.NoError(t, err)

	_, err = LoadFromDirectory(filePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a directory")

	// Test with invalid YAML file
	invalidPath := filepath.Join(tmpDir, "invalid.yaml")
	err = os.WriteFile(invalidPath, []byte("invalid: yaml: content"), 0o644)
	require.NoError(t, err)

	_, err = LoadFromDirectory(tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load")
}
