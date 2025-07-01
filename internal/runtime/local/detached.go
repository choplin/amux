package local

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	amuxruntime "github.com/aki/amux/internal/runtime"
)

// DetachedRuntime implements the local process runtime for background/detached execution
type DetachedRuntime struct {
	baseRuntime
}

// NewDetachedRuntime creates a new detached runtime
func NewDetachedRuntime() *DetachedRuntime {
	return &DetachedRuntime{}
}

// Type returns the runtime type identifier
func (r *DetachedRuntime) Type() string {
	return "local-detached"
}

// Execute starts a new process in detached mode
func (r *DetachedRuntime) Execute(ctx context.Context, spec amuxruntime.ExecutionSpec) (amuxruntime.Process, error) {
	// Validate command
	if len(spec.Command) == 0 {
		return nil, amuxruntime.ErrInvalidCommand
	}

	// Get shell
	shell := getShell()

	// Create command without context to avoid automatic termination
	cmd := createCommand(ctx, spec, shell, false)

	// Setup command properties
	if err := setupCommand(cmd, spec); err != nil {
		return nil, err
	}

	// Configure process isolation for detached mode
	configureProcessIsolation(cmd, true)

	// Create process
	proc := createProcess(spec)
	proc.cmd = cmd

	// Create log directory for detached process
	logDir := filepath.Join(os.TempDir(), "amux-logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file for output
	logFile := filepath.Join(logDir, fmt.Sprintf("session-%s.log", proc.id))
	outFile, err := os.Create(logFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Store log file path in process
	proc.logFile = logFile

	// Redirect output to log file
	cmd.Stdout = outFile
	cmd.Stderr = outFile

	// Start the process
	if err := cmd.Start(); err != nil {
		_ = outFile.Close()
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	proc.setState(amuxruntime.StateRunning)

	// Create metadata after process starts
	proc.metadata = &Metadata{
		PID:      cmd.Process.Pid,
		Detached: true,
	}

	// Try to get PGID (might not work on all platforms)
	if isProcessGroup(cmd) {
		proc.metadata.PGID = cmd.Process.Pid
	}

	// Store process
	r.processes.Store(proc.id, proc)

	// Monitor process completion (also closes the log file)
	go func() {
		proc.monitor()
		// Close the log file when process completes
		_ = outFile.Close()
	}()

	// Detached processes don't handle context cancellation
	// They continue running even if the parent context is cancelled

	return proc, nil
}

// CaptureOutput implements runtime.OutputCapture interface
func (p *Process) CaptureOutput(lines int) ([]byte, error) {
	p.mu.RLock()
	logFile := p.logFile
	p.mu.RUnlock()

	if logFile == "" {
		return nil, fmt.Errorf("no log file available for process")
	}

	// Open the log file
	file, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Log file doesn't exist yet, return empty
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// If lines is 0, read the entire file
	if lines == 0 {
		content, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read log file: %w", err)
		}
		return content, nil
	}

	// Read last N lines
	var outputLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		outputLines = append(outputLines, scanner.Text())
		if len(outputLines) > lines {
			outputLines = outputLines[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan log file: %w", err)
	}

	return []byte(strings.Join(outputLines, "\n")), nil
}

// StreamOutput implements runtime.OutputStreamer interface
func (p *Process) StreamOutput(ctx context.Context, w io.Writer, opts amuxruntime.StreamOptions) error {
	p.mu.RLock()
	logFile := p.logFile
	p.mu.RUnlock()

	if logFile == "" {
		return fmt.Errorf("no log file available for process")
	}

	// Open the log file
	file, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Log file doesn't exist yet, wait a bit
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return p.StreamOutput(ctx, w, opts)
			}
		}
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// StreamOutput always follows the file, reading from beginning

	reader := bufio.NewReader(file)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Wait for more data
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-p.done:
						// Process completed, read any remaining data
						remaining, _ := io.ReadAll(reader)
						if len(remaining) > 0 {
							_, _ = w.Write(remaining)
						}
						return nil
					case <-time.After(100 * time.Millisecond):
						// Continue loop to check for new data
					}
					continue
				}
				return fmt.Errorf("failed to read line: %w", err)
			}

			// Write the line
			if _, err := w.Write([]byte(line)); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}
		}
	}
}
