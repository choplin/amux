//go:build windows

package local

import (
	"os/exec"
	"syscall"
)

const (
	// CREATE_NEW_PROCESS_GROUP is the Windows flag for creating a new process group
	CREATE_NEW_PROCESS_GROUP = syscall.CREATE_NEW_PROCESS_GROUP
)

// configureProcessIsolation sets up process isolation for Windows
func configureProcessIsolation(cmd *exec.Cmd, detach bool) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	if detach {
		// On Windows, hide the window and create a new process group
		cmd.SysProcAttr.HideWindow = true
		cmd.SysProcAttr.CreationFlags = CREATE_NEW_PROCESS_GROUP
	}
}
