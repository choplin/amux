// Package process provides utilities for process management
package process

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Checker defines the interface for process checking operations
type Checker interface {
	HasChildren(pid int) (bool, error)
}

// DefaultChecker implements Checker using system commands
type DefaultChecker struct{}

// HasChildren checks if a process has any child processes
func (c *DefaultChecker) HasChildren(pid int) (bool, error) {
	// Use pgrep to find child processes
	// -P flag specifies parent PID
	cmd := exec.Command("pgrep", "-P", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		// pgrep returns exit code 1 when no processes are found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("failed to check child processes: %w", err)
	}

	// If we got any output, there are child processes
	return strings.TrimSpace(string(output)) != "", nil
}

// Default is the default process checker
var Default = &DefaultChecker{}

// HasChildren is a convenience function using the default checker
func HasChildren(pid int) (bool, error) {
	return Default.HasChildren(pid)
}
