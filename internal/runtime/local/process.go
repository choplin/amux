package local

import (
	"os/exec"
)

// isProcessGroup checks if the command is configured to use a process group
func isProcessGroup(cmd *exec.Cmd) bool {
	return isProcessGroupPlatform(cmd)
}

// signalStop sends a stop signal to the process or process group
func signalStop(cmd *exec.Cmd) error {
	return signalStopPlatform(cmd)
}

// signalKill sends a kill signal to the process or process group
func signalKill(cmd *exec.Cmd) error {
	return signalKillPlatform(cmd)
}
