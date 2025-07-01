// Package runtime provides abstractions for process execution
package runtime

import (
	"context"
	"io"
	"time"
)

// OutputCapture provides output capture capabilities
type OutputCapture interface {
	Process
	// CaptureOutput captures recent output
	// lines=0 means capture based on terminal size
	CaptureOutput(lines int) ([]byte, error)
}

// OutputStreamer provides output streaming capabilities
type OutputStreamer interface {
	OutputCapture
	// StreamOutput streams output in real-time
	StreamOutput(ctx context.Context, w io.Writer, opts StreamOptions) error
}

// StreamOptions configures output streaming behavior
type StreamOptions struct {
	PollInterval time.Duration
	ClearScreen  bool // Clear screen and redraw
}

// AttachableProcess provides terminal attach capabilities
type AttachableProcess interface {
	Process
	// Attach attaches to the process terminal
	Attach() error
}

// InputSender provides input sending capabilities
type InputSender interface {
	Process
	// SendInput sends input to the process stdin
	SendInput(input string) error
}
