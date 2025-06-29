package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigWithTasks(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content string
		wantErr bool
		errMsg  string
		check   func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid config with tasks",
			content: `version: "1.0"
agents:
  claude:
    name: Claude
    runtime: local
tasks:
  - name: build
    command: go build
    description: Build the application
  - name: test
    command: go test ./...
    lifecycle: oneshot
  - name: dev
    command: npm run dev
    lifecycle: daemon`,
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Tasks, 3)
				assert.Equal(t, "build", cfg.Tasks[0].Name)
				assert.Equal(t, "test", cfg.Tasks[1].Name)
				assert.Equal(t, "dev", cfg.Tasks[2].Name)
			},
		},
		{
			name: "task with dependencies",
			content: `version: "1.0"
agents:
  claude:
    name: Claude
    runtime: local
tasks:
  - name: prepare
    command: go mod download
  - name: build
    command: go build
    depends_on:
      - prepare`,
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Tasks, 2)
				assert.Equal(t, []string{"prepare"}, cfg.Tasks[1].DependsOn)
			},
		},
		{
			name: "task with invalid dependency",
			content: `version: "1.0"
agents:
  claude:
    name: Claude
    runtime: local
tasks:
  - name: build
    command: go build
    depends_on:
      - nonexistent`,
			wantErr: true,
			errMsg:  "depends on non-existent task",
		},
		{
			name: "task with environment variables",
			content: `version: "1.0"
agents:
  claude:
    name: Claude
    runtime: local
tasks:
  - name: test
    command: go test
    env:
      GOOS: linux
      GOARCH: amd64`,
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				require.Len(t, cfg.Tasks, 1)
				assert.Equal(t, "linux", cfg.Tasks[0].Env["GOOS"])
				assert.Equal(t, "amd64", cfg.Tasks[0].Env["GOARCH"])
			},
		},
		{
			name: "task with invalid name",
			content: `version: "1.0"
agents:
  claude:
    name: Claude
    runtime: local
tasks:
  - name: ""
    command: go build`,
			wantErr: true,
			errMsg:  "length must be >= 1",
		},
		{
			name: "task with invalid lifecycle",
			content: `version: "1.0"
agents:
  claude:
    name: Claude
    runtime: local
tasks:
  - name: build
    command: go build
    lifecycle: invalid`,
			wantErr: true,
			errMsg:  "schema validation failed",
		},
		{
			name: "daemon task with timeout",
			content: `version: "1.0"
agents:
  claude:
    name: Claude
    runtime: local
tasks:
  - name: server
    command: npm run dev
    lifecycle: daemon
    timeout: 30s`,
			wantErr: true,
			errMsg:  "timeout cannot be specified for daemon tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test config
			configPath := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configPath, []byte(tt.content), 0o644)
			require.NoError(t, err)

			// Load config
			cfg, err := LoadWithValidation(configPath)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
				if tt.check != nil {
					tt.check(t, cfg)
				}
			}
		})
	}
}

func TestGetTaskManager(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test config with tasks
	configContent := `version: "1.0"
agents:
  claude:
    name: Claude
    runtime: local
tasks:
  - name: build
    command: go build
  - name: test
    command: go test ./...
  - name: lint
    command: golangci-lint run`

	// Write config file
	amuxDir := filepath.Join(tmpDir, AmuxDir)
	err := os.MkdirAll(amuxDir, 0o755)
	require.NoError(t, err)

	configPath := filepath.Join(amuxDir, ConfigFile)
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	// Create manager
	manager := NewManager(tmpDir)

	// Get task manager
	taskManager, err := manager.GetTaskManager()
	require.NoError(t, err)
	require.NotNil(t, taskManager)

	// Verify tasks were loaded
	tasks := taskManager.ListTasks()
	assert.Len(t, tasks, 3)

	// Check task names
	names := taskManager.GetTaskNames()
	assert.Contains(t, names, "build")
	assert.Contains(t, names, "test")
	assert.Contains(t, names, "lint")

	// Get specific task
	buildTask, err := taskManager.GetTask("build")
	require.NoError(t, err)
	assert.Equal(t, "build", buildTask.Name)
	assert.Equal(t, "go build", buildTask.Command)
}
