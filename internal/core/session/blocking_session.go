package session

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/logger"
	"github.com/aki/amux/internal/core/process"
	"github.com/aki/amux/internal/core/workspace"
)

// blockingSessionImpl implements a blocking command execution session
type blockingSessionImpl struct {
	info           *Info
	manager        SessionManager
	workspace      *workspace.Workspace
	agentConfig    *config.Agent
	logger         logger.Logger
	processChecker process.Checker

	// Blocking-specific fields
	cmd          *exec.Cmd
	outputConfig *OutputConfig
	outputBuffer *bytes.Buffer   // For buffer mode
	outputFile   *os.File        // For file mode
	circularBuf  *circularBuffer // For circular mode
	bufferFull   bool
	wasStopped   bool
	startTime    time.Time
	endTime      time.Time
	exitCode     int
	mu           sync.RWMutex
	outputMu     sync.Mutex // Separate mutex for output operations
}

// NewBlockingSession creates a new blocking session
func NewBlockingSession(
	info *Info,
	manager SessionManager,
	ws *workspace.Workspace,
	agentConfig *config.Agent,
	logger logger.Logger,
) (Session, error) {
	// Set default output config if not provided
	if info.OutputConfig == nil {
		info.OutputConfig = GetDefaultOutputConfig()
	}

	return &blockingSessionImpl{
		info:           info,
		manager:        manager,
		workspace:      ws,
		agentConfig:    agentConfig,
		logger:         logger,
		processChecker: &process.DefaultChecker{},
		outputConfig:   info.OutputConfig,
	}, nil
}

// Interface implementation
func (s *blockingSessionImpl) ID() string {
	return s.info.ID
}

func (s *blockingSessionImpl) WorkspaceID() string {
	return s.info.WorkspaceID
}

func (s *blockingSessionImpl) WorkspacePath() string {
	if s.workspace != nil {
		return s.workspace.Path
	}
	return ""
}

func (s *blockingSessionImpl) AgentID() string {
	return s.info.AgentID
}

func (s *blockingSessionImpl) Type() Type {
	return s.info.Type
}

func (s *blockingSessionImpl) Info() *Info {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy to avoid race conditions
	infoCopy := *s.info
	infoCopy.ExitCode = s.exitCode
	infoCopy.BufferFull = s.bufferFull

	return &infoCopy
}

func (s *blockingSessionImpl) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check workspace existence first
	if s.workspace != nil && !s.workspace.PathExists {
		return StatusOrphaned
	}

	// Check process state
	if s.cmd == nil {
		return StatusCreated
	}

	if s.cmd.ProcessState == nil {
		// Still running
		return StatusWorking
	}

	// Process finished
	if s.cmd.ProcessState.Success() {
		return StatusCompleted
	}

	// Check if manually stopped vs crashed
	if s.wasStopped {
		return StatusStopped
	}

	return StatusFailed
}

func (s *blockingSessionImpl) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd != nil {
		return fmt.Errorf("session already started")
	}

	// Validate workspace exists
	if s.workspace == nil || !s.workspace.PathExists {
		return fmt.Errorf("workspace not available")
	}

	// Set up output capture
	if err := s.setupOutputCapture(); err != nil {
		return fmt.Errorf("failed to set up output capture: %w", err)
	}

	// Create command
	s.cmd = exec.CommandContext(ctx, s.info.BlockingCommand, s.info.BlockingArgs...)
	s.cmd.Dir = s.workspace.Path

	// Set up environment
	s.cmd.Env = os.Environ()
	for k, v := range s.info.Environment {
		s.cmd.Env = append(s.cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up stdout/stderr capture
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	s.startTime = time.Now()
	s.info.PID = s.cmd.Process.Pid

	// Update session info
	now := time.Now()
	s.info.StartedAt = &now
	s.info.StatusState.Status = StatusWorking
	s.info.StatusState.StatusChangedAt = now

	// Save updated info
	if err := s.manager.Update(ctx, s.info.ID, func(info *Info) error {
		*info = *s.info
		return nil
	}); err != nil {
		s.logger.Error("Failed to update session info", "error", err)
	}

	// Start output capture in a single goroutine
	go s.captureOutputMultiplexed(stdout, stderr)

	// Start a goroutine to wait for process completion
	go s.waitForCompletion() //nolint:contextcheck // Process lifetime managed by cmd.Wait

	s.logger.Info("Started blocking session",
		"id", s.info.ID,
		"command", s.info.BlockingCommand,
		"args", s.info.BlockingArgs,
		"pid", s.info.PID,
	)

	return nil
}

func (s *blockingSessionImpl) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd == nil || s.cmd.ProcessState != nil {
		return fmt.Errorf("session not running")
	}

	s.wasStopped = true

	// First try graceful termination with SIGTERM
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, try SIGKILL
		if err := s.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop process: %w", err)
		}
	}

	// Wait for up to 5 seconds for graceful shutdown
	done := make(chan struct{})
	go func() {
		_ = s.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Process stopped gracefully
	case <-time.After(5 * time.Second):
		// Force kill if still running
		if err := s.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process after timeout: %w", err)
		}
	}

	// Update status
	now := time.Now()
	s.info.StoppedAt = &now
	s.info.StatusState.Status = StatusStopped
	s.info.StatusState.StatusChangedAt = now

	// Clean up output resources
	s.cleanupOutput()

	// Save updated info
	if err := s.manager.Update(ctx, s.info.ID, func(info *Info) error {
		*info = *s.info
		return nil
	}); err != nil {
		s.logger.Error("Failed to update session info", "error", err)
	}

	s.logger.Info("Stopped blocking session", "id", s.info.ID)

	return nil
}

// Output capture methods

func (s *blockingSessionImpl) setupOutputCapture() error {
	switch s.outputConfig.Mode {
	case OutputModeBuffer:
		s.outputBuffer = bytes.NewBuffer(nil)

	case OutputModeFile:
		filePath := s.outputConfig.FilePath
		if filePath == "" {
			filePath = filepath.Join(s.info.StoragePath, "output.log")
		}
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		s.outputFile = file

	case OutputModeCircular:
		s.circularBuf = newCircularBuffer(s.outputConfig.BufferSize)
	}

	return nil
}

func (s *blockingSessionImpl) captureOutputMultiplexed(stdout, stderr io.Reader) {
	// Combine stdout and stderr into a single reader
	multiReader := io.MultiReader(stdout, stderr)

	buf := make([]byte, 4096)
	for {
		n, err := multiReader.Read(buf)
		if n > 0 {
			s.outputMu.Lock()
			switch s.outputConfig.Mode {
			case OutputModeBuffer:
				if int64(s.outputBuffer.Len()+n) > s.outputConfig.BufferSize {
					s.bufferFull = true
					s.outputMu.Unlock()
					return // Stop capturing
				}
				s.outputBuffer.Write(buf[:n])

			case OutputModeFile:
				if _, err := s.outputFile.Write(buf[:n]); err != nil {
					s.logger.Error("Failed to write to output file", "error", err)
				}

			case OutputModeCircular:
				_, _ = s.circularBuf.Write(buf[:n])
			}
			s.outputMu.Unlock()
		}

		if err != nil {
			if err != io.EOF {
				s.logger.Error("Error reading output", "error", err)
			}
			break
		}
	}
}

func (s *blockingSessionImpl) cleanupOutput() {
	if s.outputFile != nil {
		_ = s.outputFile.Close()
		s.outputFile = nil
	}
}

func (s *blockingSessionImpl) waitForCompletion() {
	// Wait for the command to complete
	err := s.cmd.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.endTime = time.Now()
	endTime := s.endTime
	s.info.StoppedAt = &endTime

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			s.exitCode = exitErr.ExitCode()
			s.info.ExitCode = s.exitCode
			if !s.wasStopped {
				s.info.StatusState.Status = StatusFailed
				s.info.Error = fmt.Sprintf("command exited with code %d", s.exitCode)
			}
		} else {
			s.info.StatusState.Status = StatusFailed
			s.info.Error = err.Error()
		}
	} else {
		s.exitCode = 0
		s.info.ExitCode = 0
		s.info.StatusState.Status = StatusCompleted
	}

	s.info.StatusState.StatusChangedAt = s.endTime

	// Clean up output resources
	s.cleanupOutput()

	// Save updated info
	updateCtx := context.Background()
	if err := s.manager.Update(updateCtx, s.info.ID, func(info *Info) error {
		*info = *s.info
		return nil
	}); err != nil {
		s.logger.Error("Failed to update session info", "error", err)
	}
}

// GetOutput returns the captured output
func (s *blockingSessionImpl) GetOutput(maxLines int) ([]byte, error) {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()

	switch s.outputConfig.Mode {
	case OutputModeBuffer:
		if s.outputBuffer == nil {
			return nil, fmt.Errorf("output buffer not initialized")
		}
		data := s.outputBuffer.Bytes()
		if s.bufferFull {
			// Prepend warning about buffer being full
			warning := []byte("[WARNING: Output buffer full, some output was discarded]\n")
			data = append(warning, data...)
		}
		return data, nil

	case OutputModeFile:
		if s.outputConfig.FilePath == "" {
			s.outputConfig.FilePath = filepath.Join(s.info.StoragePath, "output.log")
		}
		// For file mode, read last N lines
		return readLastLines(s.outputConfig.FilePath, maxLines)

	case OutputModeCircular:
		if s.circularBuf == nil {
			return nil, fmt.Errorf("circular buffer not initialized")
		}
		return s.circularBuf.Bytes(), nil

	default:
		return nil, fmt.Errorf("unknown output mode: %s", s.outputConfig.Mode)
	}
}

// readLastLines reads the last N lines from a file
func readLastLines(filePath string, maxLines int) ([]byte, error) {
	if maxLines <= 0 {
		// Read entire file
		return os.ReadFile(filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output file: %w", err)
	}

	// Split into lines and get last N
	lines := bytes.Split(data, []byte("\n"))
	if len(lines) <= maxLines {
		return data, nil
	}

	startIdx := len(lines) - maxLines
	result := bytes.Join(lines[startIdx:], []byte("\n"))
	return result, nil
}
