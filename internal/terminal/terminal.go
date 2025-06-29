// Package terminal provides terminal-related utility functions
package terminal

import (
	"os"

	"github.com/charmbracelet/x/term"
)

// GetSize returns the current terminal dimensions or defaults
func GetSize() (width, height int) {
	// Default dimensions
	width, height = 120, 40

	// Try to get terminal size from stdout
	if w, h, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 && h > 0 {
		width, height = w, h
		return
	}

	// Try stderr as fallback
	if w, h, err := term.GetSize(os.Stderr.Fd()); err == nil && w > 0 && h > 0 {
		width, height = w, h
	}
	return
}
