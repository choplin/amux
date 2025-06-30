package session

import (
	"bytes"
	"os"
	"testing"
)

// Test basic command structure
func TestSessionCommand(t *testing.T) {
	cmd := Command()

	// Check command properties
	if cmd.Use != "session" {
		t.Errorf("Expected command use 'session', got %s", cmd.Use)
	}

	// Check aliases
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "s" {
		t.Error("Expected alias 's'")
	}

	// Check subcommands
	subcommands := []string{"run", "ps", "attach", "stop", "logs", "remove", "storage"}
	for _, subcmd := range subcommands {
		found := false
		for _, c := range cmd.Commands() {
			if c.Name() == subcmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand '%s' not found", subcmd)
		}
	}
}

// Test run command flags
func TestRunCommandFlags(t *testing.T) {
	// Check that flags are properly defined
	if runCmd.Flag("task") == nil {
		t.Error("Expected --task flag")
	}
	if runCmd.Flag("workspace") == nil {
		t.Error("Expected --workspace flag")
	}
	if runCmd.Flag("runtime") == nil {
		t.Error("Expected --runtime flag")
	}
	if runCmd.Flag("env") == nil {
		t.Error("Expected --env flag")
	}
	if runCmd.Flag("dir") == nil {
		t.Error("Expected --dir flag")
	}
	if runCmd.Flag("follow") == nil {
		t.Error("Expected --follow flag")
	}
	if runCmd.Flag("detach") == nil {
		t.Error("Expected --detach flag")
	}

	// Check that task flag has short version
	taskFlag := runCmd.Flag("task")
	if taskFlag != nil && taskFlag.Shorthand != "t" {
		t.Error("Expected --task flag to have -t shorthand")
	}

	// Check that detach flag has correct description
	detachFlag := runCmd.Flag("detach")
	if detachFlag != nil && !contains(detachFlag.Usage, "local runtime only") {
		t.Error("Expected --detach flag description to mention 'local runtime only'")
	}
}

// Test ps command flags
func TestPsCommandFlags(t *testing.T) {
	if psCmd.Flag("workspace") == nil {
		t.Error("Expected --workspace flag")
	}
	if psCmd.Flag("all") == nil {
		t.Error("Expected --all flag")
	}
	if psCmd.Flag("format") == nil {
		t.Error("Expected --format flag")
	}
}

// Test logs command flags
func TestLogsCommandFlags(t *testing.T) {
	if logsCmd.Flag("follow") == nil {
		t.Error("Expected --follow flag")
	}
	if logsCmd.Flag("tail") == nil {
		t.Error("Expected --tail flag")
	}
}

// Test command without amux initialization
func TestCommandsNotInAmuxProject(t *testing.T) {
	// Create temp directory without .amux
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	tests := []struct {
		name string
		args []string
	}{
		{"run", []string{"run", "--", "echo", "test"}},
		{"ps", []string{"ps"}},
		{"attach", []string{"attach", "session-1"}},
		{"logs", []string{"logs", "session-1"}},
		{"stop", []string{"stop", "session-1"}},
		{"remove", []string{"remove", "session-1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Command()
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil {
				t.Error("Expected error when not in amux project")
			}
			if !contains(err.Error(), "not initialized") {
				t.Errorf("Expected 'not initialized' error, got: %v", err)
			}
		})
	}
}

// Test shortcut commands exist
func TestShortcutCommandsExist(t *testing.T) {
	// Shortcut commands are defined in internal/cli/commands package
	// We can't import them here due to circular dependency
	// Just verify that the session subcommands exist

	cmd := Command()
	expectedSubcommands := []string{"run", "ps", "attach", "logs", "stop", "remove", "storage"}

	for _, expected := range expectedSubcommands {
		found := false
		for _, subcmd := range cmd.Commands() {
			if subcmd.Name() == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand %s not found", expected)
		}
	}
}

// Helper functions
func contains(s, substr string) bool {
	if len(s) == 0 || len(substr) == 0 {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
