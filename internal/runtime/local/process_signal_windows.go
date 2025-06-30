//go:build windows

package local

import (
	"fmt"
	"os/exec"
)

// isProcessGroupPlatform checks if the command is configured to use a process group on Windows
func isProcessGroupPlatform(cmd *exec.Cmd) bool {
	// Windows uses CREATE_NEW_PROCESS_GROUP flag instead of Setpgid
	return cmd.SysProcAttr != nil && cmd.SysProcAttr.CreationFlags&CREATE_NEW_PROCESS_GROUP != 0
}

// signalStopPlatform sends a stop signal to the process on Windows
func signalStopPlatform(cmd *exec.Cmd) error {
	// Windows doesn't have SIGTERM, use Kill directly
	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to stop process: %w", err)
	}
	return nil
}

// signalKillPlatform sends a kill signal to the process on Windows
func signalKillPlatform(cmd *exec.Cmd) error {
	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}
	return nil
}
