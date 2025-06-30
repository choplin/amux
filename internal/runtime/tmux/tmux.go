// Package tmux provides a runtime implementation that uses tmux for terminal multiplexing.
package tmux

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/aki/amux/internal/runtime"
)

// Runtime implements the tmux-based process runtime
type Runtime struct {
	executable string   // tmux binary path
	baseDir    string   // base directory for sockets
	processes  sync.Map // map[string]*Process
}

// New creates a new tmux runtime
func New(baseDir string) (*Runtime, error) {
	// Find tmux executable
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux not found in PATH: %w", err)
	}

	// Create base directory if needed
	if baseDir == "" {
		baseDir = filepath.Join(os.TempDir(), "amux-tmux")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &Runtime{
		executable: tmuxPath,
		baseDir:    baseDir,
	}, nil
}

// Type returns the runtime type identifier
func (r *Runtime) Type() string {
	return "tmux"
}

// Execute starts a new process in the tmux runtime
func (r *Runtime) Execute(ctx context.Context, spec runtime.ExecutionSpec) (runtime.Process, error) {
	// Validate command
	if len(spec.Command) == 0 {
		return nil, runtime.ErrInvalidCommand
	}

	// Get options with defaults
	opts, _ := spec.Options.(Options)
	if opts.SessionName == "" {
		opts.SessionName = fmt.Sprintf("amux-%s", uuid.New().String()[:8])
	}
	if opts.WindowName == "" {
		opts.WindowName = "amux"
	}
	if opts.OutputHistory == 0 {
		opts.OutputHistory = 10000
	}

	// Create process
	proc := &Process{
		id:          uuid.New().String(),
		sessionName: opts.SessionName,
		spec:        spec,
		state:       runtime.StateStarting,
		startTime:   time.Now(),
		opts:        opts,
		runtime:     r,
		done:        make(chan struct{}),
	}

	// Build tmux command
	args := []string{
		"new-session",
		"-d", // detached
		"-s", opts.SessionName,
		"-n", opts.WindowName,
	}

	// Add socket path if specified
	if opts.SocketPath != "" {
		args = append([]string{"-S", opts.SocketPath}, args...)
	}

	// Set working directory
	if spec.WorkingDir != "" {
		if _, err := os.Stat(spec.WorkingDir); err != nil {
			return nil, fmt.Errorf("working directory does not exist: %w", err)
		}
		args = append(args, "-c", spec.WorkingDir)
	}

	// Set environment variables
	for k, v := range spec.Environment {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	// Set remain-on-exit if requested
	if opts.RemainOnExit {
		args = append(args, "-x") // remain-on-exit
	}

	// Create session
	cmd := exec.CommandContext(ctx, r.executable, args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Send the actual command to execute
	commandStr := strings.Join(spec.Command, " ")
	sendArgs := []string{"send-keys", "-t", opts.SessionName, commandStr, "Enter"}
	if opts.SocketPath != "" {
		sendArgs = append([]string{"-S", opts.SocketPath}, sendArgs...)
	}
	sendCmd := exec.CommandContext(ctx, r.executable, sendArgs...)
	if err := sendCmd.Run(); err != nil {
		// Clean up session
		_ = r.killSession(opts.SocketPath, opts.SessionName)
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	proc.setState(runtime.StateRunning)

	// Store process
	r.processes.Store(proc.id, proc)

	// Monitor process
	go proc.monitor(ctx)

	return proc, nil
}

// Find locates an existing process by ID
func (r *Runtime) Find(ctx context.Context, id string) (runtime.Process, error) {
	if proc, ok := r.processes.Load(id); ok {
		return proc.(*Process), nil
	}
	return nil, runtime.ErrProcessNotFound
}

// List returns all processes managed by this runtime
func (r *Runtime) List(ctx context.Context) ([]runtime.Process, error) {
	var processes []runtime.Process
	r.processes.Range(func(key, value interface{}) bool {
		processes = append(processes, value.(*Process))
		return true
	})
	return processes, nil
}

// Validate checks if this runtime is properly configured and available
func (r *Runtime) Validate() error {
	// Check tmux version
	cmd := exec.Command(r.executable, "-V")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get tmux version: %w", err)
	}

	// Parse version (e.g., "tmux 3.3a")
	version := strings.TrimSpace(string(output))
	parts := strings.Fields(version)
	if len(parts) < 2 {
		return fmt.Errorf("unexpected tmux version format: %s", version)
	}

	// TODO: Check minimum version requirements
	return nil
}

// killSession kills a tmux session
func (r *Runtime) killSession(socketPath, sessionName string) error {
	args := []string{"kill-session", "-t", sessionName}
	if socketPath != "" {
		args = append([]string{"-S", socketPath}, args...)
	}
	cmd := exec.Command(r.executable, args...)
	return cmd.Run()
}

// tmuxCmd builds tmux command with optional socket path
func (r *Runtime) tmuxCmd(socketPath string, args ...string) *exec.Cmd {
	if socketPath != "" {
		args = append([]string{"-S", socketPath}, args...)
	}
	return exec.Command(r.executable, args...)
}

// Process represents a tmux process
type Process struct {
	id          string
	sessionName string
	spec        runtime.ExecutionSpec
	state       runtime.ProcessState
	startTime   time.Time
	opts        Options
	runtime     *Runtime
	mu          sync.RWMutex
	done        chan struct{}
	doneOnce    sync.Once
	exitCode    int
}

// ID returns the unique identifier for this process
func (p *Process) ID() string {
	return p.id
}

// State returns the current state of the process
func (p *Process) State() runtime.ProcessState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// Wait blocks until the process completes
func (p *Process) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		// For tmux, we don't have reliable exit codes
		// Just return nil if the process completed
		return nil
	}
}

// Stop gracefully stops the process (SIGTERM)
func (p *Process) Stop(ctx context.Context) error {
	p.mu.Lock()
	if p.state != runtime.StateRunning {
		p.mu.Unlock()
		return runtime.ErrProcessAlreadyDone
	}
	p.mu.Unlock()

	// Send SIGTERM via tmux
	cmd := p.runtime.tmuxCmd(p.opts.SocketPath, "send-keys", "-t", p.sessionName, "C-c")
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send interrupt: %w", err)
	}

	// Give process time to stop gracefully
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		// Force kill if still running
		return p.Kill(ctx)
	case <-p.done:
		return nil
	}
}

// Kill forcefully terminates the process (SIGKILL)
func (p *Process) Kill(ctx context.Context) error {
	p.mu.Lock()
	if p.state != runtime.StateRunning {
		p.mu.Unlock()
		return runtime.ErrProcessAlreadyDone
	}
	p.mu.Unlock()

	// Kill the tmux session
	if err := p.runtime.killSession(p.opts.SocketPath, p.sessionName); err != nil {
		return fmt.Errorf("failed to kill session: %w", err)
	}

	// Update state
	p.setState(runtime.StateFailed)
	p.doneOnce.Do(func() {
		close(p.done)
	})

	return nil
}

// Output returns readers for stdout and stderr
func (p *Process) Output() (stdout, stderr io.Reader) {
	if !p.opts.CaptureOutput {
		return nil, nil
	}

	// Capture pane content
	output, err := p.capturePane()
	if err != nil {
		return bytes.NewReader(nil), bytes.NewReader(nil)
	}

	// Tmux doesn't separate stdout/stderr, return all as stdout
	return bytes.NewReader([]byte(output)), bytes.NewReader(nil)
}

// ExitCode returns the exit code (valid after process completes)
func (p *Process) ExitCode() (int, error) {
	select {
	case <-p.done:
		p.mu.RLock()
		code := p.exitCode
		p.mu.RUnlock()
		return code, nil
	default:
		return -1, fmt.Errorf("process still running")
	}
}

// StartTime returns when the process was started
func (p *Process) StartTime() time.Time {
	return p.startTime
}

// Metadata returns runtime-specific metadata
func (p *Process) Metadata() runtime.Metadata {
	return &Metadata{
		SessionName: p.sessionName,
		WindowName:  p.opts.WindowName,
		// PaneID could be retrieved dynamically if needed, but for now leave empty
	}
}

// Attach creates a new client attached to the tmux session
func (p *Process) Attach(ctx context.Context) error {
	p.mu.RLock()
	state := p.state
	p.mu.RUnlock()

	if state != runtime.StateRunning {
		return fmt.Errorf("cannot attach to process in state: %s", state)
	}

	// Check if session still exists
	if !p.sessionExists() {
		return fmt.Errorf("tmux session no longer exists")
	}

	// Create attach command
	cmd := p.runtime.tmuxCmd(p.opts.SocketPath, "attach-session", "-t", p.sessionName)
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// setState updates the process state
func (p *Process) setState(state runtime.ProcessState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.state = state
}

// sessionExists checks if the tmux session still exists
func (p *Process) sessionExists() bool {
	cmd := p.runtime.tmuxCmd(p.opts.SocketPath, "has-session", "-t", p.sessionName)
	return cmd.Run() == nil
}

// capturePane captures the pane content
func (p *Process) capturePane() (string, error) {
	// Use -S to capture from the beginning of the history
	// and -E to capture to the end
	cmd := p.runtime.tmuxCmd(p.opts.SocketPath, "capture-pane", "-t", p.sessionName, "-p", "-S", "-", "-E", "-")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to capture pane: %w", err)
	}

	return string(output), nil
}

// isPaneDead checks if the pane is dead
func (p *Process) isPaneDead() (bool, error) {
	cmd := p.runtime.tmuxCmd(p.opts.SocketPath, "list-panes", "-t", p.sessionName, "-F", "#{pane_dead}")
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "session not found") {
			return true, nil
		}
		return false, err
	}

	return strings.TrimSpace(string(output)) == "1", nil
}

// getExitStatus tries to get the exit status from the pane
func (p *Process) getExitStatus() int {
	// This is a limitation of tmux - it doesn't directly expose exit codes
	// We assume success (0) for dead panes unless we know otherwise
	return 0
}

// monitor watches the tmux session for completion
func (p *Process) monitor(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.setState(runtime.StateFailed)
			p.doneOnce.Do(func() {
				close(p.done)
			})
			return
		case <-ticker.C:
			// Check if session still exists
			if !p.sessionExists() {
				p.mu.Lock()
				p.state = runtime.StateStopped
				p.exitCode = 1 // Assume failure if session disappeared
				p.mu.Unlock()
				p.doneOnce.Do(func() {
					close(p.done)
				})
				return
			}

			// Check if pane is dead
			dead, err := p.isPaneDead()
			if err == nil && dead {
				p.mu.Lock()
				p.state = runtime.StateStopped
				p.exitCode = p.getExitStatus()
				p.mu.Unlock()

				// Kill session if remain-on-exit is not set
				if !p.opts.RemainOnExit {
					_ = p.runtime.killSession(p.opts.SocketPath, p.sessionName)
				}

				p.doneOnce.Do(func() {
					close(p.done)
				})
				return
			}
		}
	}
}

// Options implements runtime.RuntimeOptions for tmux processes
type Options struct {
	SessionName   string // Tmux session name (generated if empty)
	WindowName    string // Window name (default: "amux")
	SocketPath    string // Custom socket path (generated if empty)
	RemainOnExit  bool   // Keep pane open after process exits
	CaptureOutput bool   // Capture pane output
	OutputHistory int    // Lines of history to keep (default: 10000)
}

// IsRuntimeOptions implements the RuntimeOptions interface
func (Options) IsRuntimeOptions() {}
