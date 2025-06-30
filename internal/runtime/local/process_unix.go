//go:build !windows

package local

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

// configureProcessIsolation sets up process isolation for Unix-like systems
func configureProcessIsolation(cmd *exec.Cmd, detach bool) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	if detach {
		// Create new process group (detach from parent)
		cmd.SysProcAttr.Setpgid = true
		// Create new session (detach from controlling terminal)
		// Note: Setsid is not supported on macOS when running under certain conditions
		// (e.g., when the parent process is not a session leader)
		// Also, some CI environments (like GitHub Actions) may restrict Setsid
		// So we only set it on Linux when not in CI
		if runtime.GOOS == "linux" && os.Getenv("CI") == "" {
			cmd.SysProcAttr.Setsid = true
		}
	} else {
		// For foreground execution, still create a new process group
		// This allows proper signal handling and process management
		cmd.SysProcAttr.Setpgid = true
	}
}
