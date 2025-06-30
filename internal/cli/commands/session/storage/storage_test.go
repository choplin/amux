package storage

import (
	"testing"

	"github.com/spf13/cobra"
)

// Test basic command structure
func TestStorageCommand(t *testing.T) {
	cmd := Command()

	// Check command properties
	if cmd.Use != "storage" {
		t.Errorf("Expected command use 'storage', got %s", cmd.Use)
	}

	// Check subcommands
	subcommands := []string{"list", "read", "write"}
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

// Test list command structure
func TestListCommand(t *testing.T) {
	if listCmd.Use != "list <session-id>" {
		t.Errorf("Expected use 'list <session-id>', got %s", listCmd.Use)
	}
	if listCmd.Args == nil {
		t.Error("Expected Args to be set")
	}
}

// Test read command structure
func TestReadCommand(t *testing.T) {
	if readCmd.Use != "read <session-id> <filename>" {
		t.Errorf("Expected use 'read <session-id> <filename>', got %s", readCmd.Use)
	}
	if readCmd.Args == nil {
		t.Error("Expected Args to be set")
	}
}

// Test write command structure
func TestWriteCommand(t *testing.T) {
	if writeCmd.Use != "write <session-id> <filename>" {
		t.Errorf("Expected use 'write <session-id> <filename>', got %s", writeCmd.Use)
	}
	if writeCmd.Args == nil {
		t.Error("Expected Args to be set")
	}
}

// Test that commands require exact args
func TestCommandArgs(t *testing.T) {
	tests := []struct {
		cmd  *cobra.Command
		args int
	}{
		{listCmd, 1},
		{readCmd, 2},
		{writeCmd, 2},
	}

	for _, tt := range tests {
		t.Run(tt.cmd.Name(), func(t *testing.T) {
			// Try with no args
			err := tt.cmd.Args(tt.cmd, []string{})
			if err == nil {
				t.Error("Expected error with no args")
			}

			// Try with correct args
			args := make([]string, tt.args)
			for i := range args {
				args[i] = "arg"
			}
			err = tt.cmd.Args(tt.cmd, args)
			if err != nil {
				t.Errorf("Expected no error with %d args, got: %v", tt.args, err)
			}
		})
	}
}
