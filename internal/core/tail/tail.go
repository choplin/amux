// Package tail provides log tailing functionality for sessions.
package tail

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aki/amux/internal/core/session"
)

// Options configures the tail behavior
type Options struct {
	// PollInterval is how often to check for new output
	PollInterval time.Duration
	// Writer is where to write the output
	Writer io.Writer
	// RefreshMode refreshes the entire display instead of appending
	RefreshMode bool
}

// DefaultOptions returns default tail options
func DefaultOptions() Options {
	return Options{
		PollInterval: 500 * time.Millisecond,
		RefreshMode:  true, // Default to refresh mode for better AI agent output
	}
}

// Tailer handles streaming session output
type Tailer struct {
	session session.Session
	opts    Options
}

// New creates a new Tailer for a session
func New(sess session.Session, opts Options) *Tailer {
	if opts.PollInterval == 0 {
		opts.PollInterval = DefaultOptions().PollInterval
	}
	return &Tailer{
		session: sess,
		opts:    opts,
	}
}

// Follow continuously streams session output until the context is cancelled
// or the session stops running
func (t *Tailer) Follow(ctx context.Context) error {
	ticker := time.NewTicker(t.opts.PollInterval)
	defer ticker.Stop()

	// Track last output for append mode
	var lastOutput []byte

	// Get initial output
	output, err := t.session.GetOutput()
	if err != nil {
		return fmt.Errorf("failed to get initial output: %w", err)
	}

	// Write initial output if any
	if len(output) > 0 && t.opts.Writer != nil {
		if t.opts.RefreshMode {
			// Clear screen and move cursor to top
			t.clearScreen()
		}
		if _, err := t.opts.Writer.Write(output); err != nil {
			return fmt.Errorf("failed to write initial output: %w", err)
		}
	}
	lastOutput = output

	// Stream output
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check if session is still running
			if t.session.Status() != session.StatusRunning {
				return nil // Session ended normally
			}

			// Get current output
			output, err := t.session.GetOutput()
			if err != nil {
				// Session might have ended, check status
				if t.session.Status() != session.StatusRunning {
					return nil // Session ended normally
				}
				return fmt.Errorf("failed to get output: %w", err)
			}

			if t.opts.RefreshMode {
				// In refresh mode, redraw entire output when it changes
				// Compare content, not just length, to detect all changes
				if !bytes.Equal(output, lastOutput) && t.opts.Writer != nil {
					// Clear screen and redraw
					t.clearScreen()
					if _, err := t.opts.Writer.Write(output); err != nil {
						return fmt.Errorf("failed to write output: %w", err)
					}
				}
			} else {
				// In append mode, only write new content
				if len(output) > len(lastOutput) {
					newContent := output[len(lastOutput):]
					if t.opts.Writer != nil {
						if _, err := t.opts.Writer.Write(newContent); err != nil {
							return fmt.Errorf("failed to write output: %w", err)
						}
					}
				}
			}
			lastOutput = output
		}
	}
}

// clearScreen clears the terminal screen using ANSI escape codes
func (t *Tailer) clearScreen() {
	if t.opts.Writer != nil {
		// Clear screen and move cursor to top-left
		t.opts.Writer.Write([]byte("\033[2J\033[H"))
	}
}

// FollowFunc is a convenience function that follows a session with default options
func FollowFunc(ctx context.Context, sess session.Session, w io.Writer) error {
	opts := DefaultOptions()
	opts.Writer = w
	tailer := New(sess, opts)
	return tailer.Follow(ctx)
}
