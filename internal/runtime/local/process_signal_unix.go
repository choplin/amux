//go:build !windows

package local

import (
	"fmt"
	"os/exec"
	"syscall"
)

// isProcessGroupPlatform checks if the command is configured to use a process group on Unix
func isProcessGroupPlatform(cmd *exec.Cmd) bool {
	return cmd.SysProcAttr != nil && cmd.SysProcAttr.Setpgid
}

// signalStopPlatform sends SIGTERM to the process or process group on Unix
func signalStopPlatform(cmd *exec.Cmd) error {
	if isProcessGroupPlatform(cmd) {
		// Send signal to process group (negative PID)
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM to process group: %w", err)
		}
	} else {
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM: %w", err)
		}
	}
	return nil
}

// signalKillPlatform sends SIGKILL to the process or process group on Unix
func signalKillPlatform(cmd *exec.Cmd) error {
	if isProcessGroupPlatform(cmd) {
		// Kill process group (negative PID)
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			return fmt.Errorf("failed to kill process group: %w", err)
		}
	} else {
		if err := cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}
	return nil
}
