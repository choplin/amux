// Package proxy provides process I/O proxying and monitoring capabilities
package proxy

import (
	"bytes"
	"container/ring"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// Status represents the status information that is periodically written
type Status struct {
	RunID     int       `yaml:"run_id"`
	PID       int       `yaml:"pid"`
	Status    string    `yaml:"status"`
	ExitCode  int       `yaml:"exit_code,omitempty"`
	StartedAt time.Time `yaml:"started_at"`
	EndedAt   time.Time `yaml:"ended_at,omitempty"`
}

// Options configures the proxy behavior
type Options struct {
	SessionDir string   // Directory to store run-specific data
	StatusPath string   // Path to status file
	LogPath    string   // Path to log file (empty if logging disabled)
	SocketPath string   // Unix socket path for output streaming
	Command    []string // Command to execute
}

// BuildProxyCommand builds command arguments for running amux proxy
// Returns the arguments to pass to exec.Command (without the binary path)
func BuildProxyCommand(sessionID string, command []string, enableLog bool) ([]string, error) {
	// Find amux binary
	amuxBin, err := findAmuxBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to find amux binary: %w", err)
	}

	// Determine paths based on session ID
	amuxDir := os.Getenv("AMUX_DIR")
	if amuxDir == "" {
		// Try to get from current working directory
		if cwd, err := os.Getwd(); err == nil {
			amuxDir = filepath.Join(cwd, ".amux")
		} else {
			amuxDir = ".amux"
		}
	}

	// Session directory
	sessionDir := filepath.Join(amuxDir, "sessions", sessionID)

	// Status file path
	statusPath := filepath.Join(sessionDir, "status.yaml")

	// Socket path (in temp directory for shorter path)
	tmpDir := os.Getenv("TMPDIR")
	if tmpDir == "" {
		tmpDir = "/tmp"
	}
	socketPath := filepath.Join(tmpDir, fmt.Sprintf("amux-%s.sock", sessionID))

	// Build proxy command arguments
	args := []string{
		amuxBin,
		"proxy",
		"--session-dir", sessionDir,
		"--status-path", statusPath,
		"--socket-path", socketPath,
	}

	// Add log path if logging is enabled
	if enableLog {
		// Pass directory, proxy will create run-specific log files
		logPath := sessionDir + "/"
		args = append(args, "--log-path", logPath)
	}

	args = append(args, "--")
	args = append(args, command...)

	return args, nil
}

// GetShell returns the appropriate shell for the current platform
func GetShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		if runtime.GOOS == "windows" {
			shell = "cmd"
		} else {
			shell = "/bin/sh"
		}
	}
	return shell
}

// findAmuxBinary tries to locate the amux binary
func findAmuxBinary() (string, error) {
	// 1. Check AMUX_BIN environment variable
	if amuxBin := os.Getenv("AMUX_BIN"); amuxBin != "" {
		if _, err := os.Stat(amuxBin); err == nil {
			return amuxBin, nil
		}
	}

	// 2. Check if amux is in PATH
	if amuxPath, err := exec.LookPath("amux"); err == nil {
		return amuxPath, nil
	}

	// 3. Not found
	return "", fmt.Errorf("amux binary not found in PATH or AMUX_BIN")
}

// Proxy manages process I/O proxying and monitoring
type Proxy struct {
	opts       Options
	status     *Status
	ringBuffer *ring.Ring
	bufferMu   sync.RWMutex
	clients    map[net.Conn]struct{}
	clientsMu  sync.RWMutex
	listener   net.Listener
}

// New creates a new proxy instance
func New(opts Options) (*Proxy, error) {
	// Validate required fields
	if opts.SessionDir == "" {
		return nil, fmt.Errorf("session directory is required")
	}
	if opts.StatusPath == "" {
		return nil, fmt.Errorf("status path is required")
	}
	if opts.SocketPath == "" {
		return nil, fmt.Errorf("socket path is required")
	}
	if len(opts.Command) == 0 {
		return nil, fmt.Errorf("command is required")
	}

	// Create ring buffer (50KB / ~50 bytes per line = ~1000 lines)
	ringSize := 1000
	p := &Proxy{
		opts:       opts,
		ringBuffer: ring.New(ringSize),
		clients:    make(map[net.Conn]struct{}),
	}

	return p, nil
}

// Run executes the proxied command
func (p *Proxy) Run() error {
	// Ensure session directory exists
	if err := os.MkdirAll(p.opts.SessionDir, 0o755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Read current status to get run_id
	var currentRunID int
	if data, err := os.ReadFile(p.opts.StatusPath); err == nil {
		var status Status
		if err := yaml.Unmarshal(data, &status); err == nil {
			currentRunID = status.RunID
		}
	}

	// Next run ID
	nextRunID := currentRunID + 1

	// Create run directory
	runDir := filepath.Join(p.opts.SessionDir, fmt.Sprintf("%d", nextRunID))
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fmt.Errorf("failed to create run directory: %w", err)
	}

	// Open log file if path is provided
	var logFile *os.File
	if p.opts.LogPath != "" {
		var err error
		// If LogPath ends with "/" or is a directory, create console.log in run directory
		logPath := p.opts.LogPath
		if strings.HasSuffix(logPath, "/") || strings.HasSuffix(logPath, string(os.PathSeparator)) {
			logPath = filepath.Join(runDir, "console.log")
		} else if info, err := os.Stat(logPath); err == nil && info.IsDir() {
			logPath = filepath.Join(runDir, "console.log")
		}
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		defer func() { _ = logFile.Close() }()
	}

	// Start Unix socket server if socket path provided
	if p.opts.SocketPath != "" {
		// Remove existing socket file
		_ = os.Remove(p.opts.SocketPath)

		// Try to use relative path if absolute path is too long
		socketPath := p.opts.SocketPath
		if len(socketPath) > 100 { // Leave some margin for safety
			// Try relative path from current directory
			cwd, _ := os.Getwd()
			if rel, err := filepath.Rel(cwd, socketPath); err == nil && len(rel) < len(socketPath) {
				socketPath = rel
			}
		}

		// Create socket
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			return fmt.Errorf("failed to create unix socket: %w", err)
		}
		p.listener = listener
		defer func() { _ = listener.Close() }()

		// Start accepting connections
		go p.acceptConnections()
	}

	// Create the command
	cmd := exec.Command(p.opts.Command[0], p.opts.Command[1:]...)

	// Set up stdin passthrough
	cmd.Stdin = os.Stdin

	// Inherit environment and working directory
	cmd.Env = os.Environ()

	// Set up stdout/stderr capture
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Initialize status
	p.status = &Status{
		RunID:     nextRunID,
		PID:       cmd.Process.Pid,
		Status:    "running",
		StartedAt: time.Now(),
	}

	// Write initial status
	if err := p.writeStatus(); err != nil {
		return fmt.Errorf("failed to write initial status: %w", err)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start I/O copying
	var wg sync.WaitGroup
	wg.Add(2)

	// Copy stdout
	go func() {
		defer wg.Done()
		p.copyOutput(os.Stdout, stdout, logFile)
	}()

	// Copy stderr
	go func() {
		defer wg.Done()
		p.copyOutput(os.Stderr, stderr, logFile)
	}()

	// Status update ticker
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Monitor loop
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	for {
		select {
		case <-ticker.C:
			p.updateStatus()

		case sig := <-sigChan:
			// Forward signal to child process
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}

		case err := <-done:
			// Process exited
			wg.Wait() // Wait for I/O to finish

			// Update final status
			p.status.Status = "exited"
			p.status.EndedAt = time.Now()
			if exitErr, ok := err.(*exec.ExitError); ok {
				p.status.ExitCode = exitErr.ExitCode()
			} else if err == nil {
				p.status.ExitCode = 0
			} else {
				p.status.ExitCode = -1
			}
			_ = p.writeStatus()

			if err != nil {
				return fmt.Errorf("command failed: %w", err)
			}
			return nil
		}
	}
}

func (p *Proxy) copyOutput(dst io.Writer, src io.Reader, logFile *os.File) {
	// Buffer for reading
	buf := make([]byte, 4096)

	// Ring buffer accumulator for partial lines
	var lineBuffer []byte

	for {
		n, err := src.Read(buf)
		if n > 0 {
			data := buf[:n]

			// Write to destination (stdout/stderr)
			_, _ = dst.Write(data)

			// Write to log file if enabled
			if logFile != nil {
				_, _ = logFile.Write(data)
			}

			// Process data for ring buffer and broadcasting
			p.processDataForBuffer(data, &lineBuffer)
		}

		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Warning: read error: %v\n", err)
			}
			break
		}
	}

	// Handle any remaining data in line buffer
	if len(lineBuffer) > 0 {
		p.addToRingBuffer(lineBuffer)
		p.broadcastData(lineBuffer)
	}
}

func (p *Proxy) updateStatus() {
	// Just write the current status
	if err := p.writeStatus(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update status: %v\n", err)
	}
}

func (p *Proxy) writeStatus() error {
	data, err := yaml.Marshal(p.status)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	// Write atomically
	tmpPath := p.opts.StatusPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write status file: %w", err)
	}

	if err := os.Rename(tmpPath, p.opts.StatusPath); err != nil {
		return fmt.Errorf("failed to rename status file: %w", err)
	}

	return nil
}

// acceptConnections handles incoming socket connections
func (p *Proxy) acceptConnections() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			// Listener closed
			return
		}

		// Add client
		p.clientsMu.Lock()
		p.clients[conn] = struct{}{}
		p.clientsMu.Unlock()

		// Send ring buffer content to new client
		go p.sendBufferToClient(conn)
	}
}

// sendBufferToClient sends the current ring buffer content to a new client
func (p *Proxy) sendBufferToClient(conn net.Conn) {
	defer func() {
		// Remove client on disconnect
		p.clientsMu.Lock()
		delete(p.clients, conn)
		p.clientsMu.Unlock()
		_ = conn.Close()
	}()

	// Send current buffer content
	p.bufferMu.RLock()
	data := make([][]byte, 0)
	p.ringBuffer.Do(func(value interface{}) {
		if d, ok := value.([]byte); ok && len(d) > 0 {
			data = append(data, d)
		}
	})
	p.bufferMu.RUnlock()

	// Write data to client
	for _, d := range data {
		if _, err := conn.Write(d); err != nil {
			return
		}
	}

	// Keep connection open for future broadcasts
	select {}
}

// processDataForBuffer processes raw data, splitting on newlines for ring buffer
func (p *Proxy) processDataForBuffer(data []byte, lineBuffer *[]byte) {
	// Append data to line buffer
	*lineBuffer = append(*lineBuffer, data...)

	// Process complete lines
	for {
		idx := bytes.IndexByte(*lineBuffer, '\n')
		if idx == -1 {
			// No more complete lines
			break
		}

		// Extract line (including newline)
		line := (*lineBuffer)[:idx+1]
		*lineBuffer = (*lineBuffer)[idx+1:]

		// Add to ring buffer and broadcast
		p.addToRingBuffer(line)
		p.broadcastData(line)
	}
}

// addToRingBuffer adds data to the ring buffer
func (p *Proxy) addToRingBuffer(data []byte) {
	p.bufferMu.Lock()
	defer p.bufferMu.Unlock()

	// Store a copy of the data
	p.ringBuffer.Value = append([]byte(nil), data...)
	p.ringBuffer = p.ringBuffer.Next()
}

// broadcastData sends data to all connected clients
func (p *Proxy) broadcastData(data []byte) {
	p.clientsMu.RLock()
	defer p.clientsMu.RUnlock()

	for conn := range p.clients {
		// Non-blocking write
		go func(c net.Conn, d []byte) {
			// Set write deadline to avoid blocking on slow clients
			_ = c.SetWriteDeadline(time.Now().Add(100 * time.Millisecond))
			if _, err := c.Write(d); err != nil {
				// Client is slow or disconnected, will be cleaned up
				_ = c.Close()
			}
		}(conn, data)
	}
}

// connectAndReadSocket connects to the socket and reads all data (for testing)
func (p *Proxy) connectAndReadSocket(ctx context.Context, w io.Writer) error {
	conn, err := net.Dial("unix", p.opts.SocketPath)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	// Copy data until context is done
	done := make(chan error)
	go func() {
		_, err := io.Copy(w, conn)
		done <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
