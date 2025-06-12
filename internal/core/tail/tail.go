// Package tail provides log tailing functionality for sessions.
package tail

import (
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
}

// DefaultOptions returns default tail options
func DefaultOptions() Options {
	return Options{
		PollInterval: 500 * time.Millisecond,
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
	// Keep track of the last output to detect changes
	var lastOutput []byte
	ticker := time.NewTicker(t.opts.PollInterval)
	defer ticker.Stop()

	// Get initial output
	output, err := t.session.GetOutput()
	if err != nil {
		return fmt.Errorf("failed to get initial output: %w", err)
	}
	lastOutput = output

	// Write initial output if any
	if len(output) > 0 && t.opts.Writer != nil {
		if _, err := t.opts.Writer.Write(output); err != nil {
			return fmt.Errorf("failed to write initial output: %w", err)
		}
	}

	// Stream new output
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

			// Check if output has changed
			if len(output) > len(lastOutput) {
				// Write only the new content
				newContent := output[len(lastOutput):]
				if t.opts.Writer != nil {
					if _, err := t.opts.Writer.Write(newContent); err != nil {
						return fmt.Errorf("failed to write output: %w", err)
					}
				}
				lastOutput = output
			}
		}
	}
}

// FollowFunc is a convenience function that follows a session with default options
func FollowFunc(ctx context.Context, sess session.Session, w io.Writer) error {
	opts := DefaultOptions()
	opts.Writer = w
	tailer := New(sess, opts)
	return tailer.Follow(ctx)
}
