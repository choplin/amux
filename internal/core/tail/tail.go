// Package tail provides log tailing functionality for sessions.
package tail

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"time"

	"github.com/aki/amux/internal/core/session"
	"github.com/charmbracelet/x/term"
)

// Options configures the tail behavior
type Options struct {
	// PollInterval is how often to check for new output
	PollInterval time.Duration
	// Writer is where to write the output
	Writer io.Writer
	// MaxLines limits the number of lines to display
	MaxLines int
}

// DefaultOptions returns default tail options
func DefaultOptions() Options {
	return Options{
		PollInterval: 1 * time.Second, // 1 second for responsive monitoring
		MaxLines:     0,               // 0 means auto-detect based on terminal size
	}
}

// Tailer handles streaming session output
type Tailer struct {
	session session.TerminalSession
	opts    Options
}

// New creates a new Tailer for a session
func New(sess session.Session, opts Options) *Tailer {
	if opts.PollInterval == 0 {
		opts.PollInterval = DefaultOptions().PollInterval
	}

	// Type assert to TerminalSession
	terminalSess, ok := sess.(session.TerminalSession)
	if !ok {
		// Return nil if session doesn't support terminal operations
		return nil
	}

	return &Tailer{
		session: terminalSess,
		opts:    opts,
	}
}

// Follow continuously streams session output until the context is cancelled
// or the session stops running
func (t *Tailer) Follow(ctx context.Context) error {
	ticker := time.NewTicker(t.opts.PollInterval)
	defer ticker.Stop()

	// Track state for optimization
	var lastHash uint32 // Hash to detect changes

	// Calculate terminal size for initial output
	maxLines := t.getTerminalLines()

	// Get initial output
	output, err := t.session.GetOutput(maxLines)
	if err != nil {
		return fmt.Errorf("failed to get initial output: %w", err)
	}

	// Display initial output
	if len(output) > 0 && t.opts.Writer != nil {
		t.clearScreen()
		if _, err := t.opts.Writer.Write(output); err != nil {
			return fmt.Errorf("failed to write initial output: %w", err)
		}
	}
	h := fnv.New32a()
	h.Write(output)
	lastHash = h.Sum32()

	// Stream output
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if session is still running
			if !t.session.Status().IsRunning() {
				return nil // Session ended normally
			}

			// Calculate terminal size (handles resize)
			maxLines := t.getTerminalLines()

			// Get current output limited to terminal size
			output, err := t.session.GetOutput(maxLines)
			if err != nil {
				// Session might have ended, check status
				if !t.session.Status().IsRunning() {
					return nil // Session ended normally
				}
				return fmt.Errorf("failed to get output: %w", err)
			}

			// Quick change detection using hash
			h := fnv.New32a()
			h.Write(output)
			currentHash := h.Sum32()
			if currentHash == lastHash {
				continue // No change, skip expensive operations
			}

			// Redraw the changed content
			if t.opts.Writer != nil {
				// Clear entire screen before redraw for clean rendering
				t.clearScreen()
				if _, err := t.opts.Writer.Write(output); err != nil {
					return fmt.Errorf("failed to write output: %w", err)
				}
			}
			lastHash = currentHash
		}
	}
}

// getTerminalLines calculates how many lines to capture based on terminal size
func (t *Tailer) getTerminalLines() int {
	// Check if MaxLines is explicitly set
	if t.opts.MaxLines > 0 {
		return t.opts.MaxLines
	}

	// Auto-detect terminal size
	_, height, err := term.GetSize(os.Stdout.Fd())
	if err != nil || height < 10 {
		return 30 // Default to 30 lines if can't detect
	}

	// Reserve 2 lines for status info
	return height - 2
}

// clearScreen clears the terminal screen using ANSI escape codes
func (t *Tailer) clearScreen() {
	if t.opts.Writer != nil {
		// Clear screen and move cursor to top-left
		_, _ = t.opts.Writer.Write([]byte("\033[2J\033[H"))
	}
}

// FollowFunc is a convenience function that follows a session with default options
func FollowFunc(ctx context.Context, sess session.Session, w io.Writer) error {
	opts := DefaultOptions()
	opts.Writer = w
	tailer := New(sess, opts)
	return tailer.Follow(ctx)
}
