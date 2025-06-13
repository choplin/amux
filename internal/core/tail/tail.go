// Package tail provides log tailing functionality for sessions.
package tail

import (
	"bytes"
	"context"
	"fmt"
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
	// RefreshMode refreshes the entire display instead of appending
	RefreshMode bool
	// MaxLines limits the number of lines to display in refresh mode
	MaxLines int
}

// DefaultOptions returns default tail options
func DefaultOptions() Options {
	return Options{
		PollInterval: 5 * time.Second, // 5 seconds is sufficient for monitoring
		RefreshMode:  true,            // Default to refresh mode for better AI agent output
		MaxLines:     0,               // 0 means auto-detect based on terminal size
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

	// Track state for optimization
	var lastOutput []byte
	var lastHash uint32 // Quick hash to detect changes

	// Get initial output
	output, err := t.session.GetOutput()
	if err != nil {
		return fmt.Errorf("failed to get initial output: %w", err)
	}

	// Process and display initial output
	displayOutput := t.processOutput(output)
	if len(displayOutput) > 0 && t.opts.Writer != nil {
		if t.opts.RefreshMode {
			t.clearScreen()
		}
		if _, err := t.opts.Writer.Write(displayOutput); err != nil {
			return fmt.Errorf("failed to write initial output: %w", err)
		}
	}
	lastOutput = output
	lastHash = quickHash(output)

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

			// Quick change detection using hash
			currentHash := quickHash(output)
			if currentHash == lastHash {
				continue // No change, skip expensive operations
			}

			if t.opts.RefreshMode {
				// Only redraw if content actually changed
				if !bytes.Equal(output, lastOutput) && t.opts.Writer != nil {
					displayOutput := t.processOutput(output)
					// Clear entire screen before redraw for clean rendering
					t.clearScreen()
					if _, err := t.opts.Writer.Write(displayOutput); err != nil {
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
			lastHash = currentHash
		}
	}
}

// processOutput processes the output based on options
func (t *Tailer) processOutput(output []byte) []byte {
	if !t.opts.RefreshMode {
		return output
	}

	// Determine number of lines to show
	maxLines := t.opts.MaxLines
	if maxLines == 0 {
		// Auto-detect terminal size
		_, height, err := term.GetSize(os.Stdout.Fd())
		if err != nil || height < 10 {
			maxLines = 10 // Minimum 10 lines
		} else {
			// Reserve 2 lines for status info
			maxLines = height - 2
		}
	}

	// Split into lines and take last N
	lines := bytes.Split(output, []byte("\n"))
	if len(lines) <= maxLines {
		return output
	}

	// Take last maxLines
	start := len(lines) - maxLines
	limited := bytes.Join(lines[start:], []byte("\n"))

	// Add indicator that output was truncated
	header := fmt.Sprintf("... (showing last %d lines) ...\n", maxLines)
	return append([]byte(header), limited...)
}

// quickHash provides a fast hash for change detection
func quickHash(data []byte) uint32 {
	var hash uint32 = 2166136261
	for _, b := range data {
		hash ^= uint32(b)
		hash *= 16777619
	}
	return hash
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
