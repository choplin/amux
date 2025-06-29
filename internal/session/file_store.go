package session

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FileStore implements Store using the filesystem
type FileStore struct {
	rootDir string
}

// NewFileStore creates a new file-based session store
func NewFileStore(rootDir string) Store {
	return &FileStore{rootDir: rootDir}
}

// sessionDir returns the directory for session data
func (s *FileStore) sessionDir() string {
	return filepath.Join(s.rootDir, "sessions")
}

// sessionFile returns the path to a session's metadata file
func (s *FileStore) sessionFile(id string) string {
	return filepath.Join(s.sessionDir(), fmt.Sprintf("session-%s.json", id))
}

// logFile returns the path to a session's log file
func (s *FileStore) logFile(id string) string {
	return filepath.Join(s.sessionDir(), fmt.Sprintf("session-%s.log", id))
}

// Save persists a session
func (s *FileStore) Save(ctx context.Context, session *Session) error {
	// Ensure directory exists
	if err := os.MkdirAll(s.sessionDir(), 0o755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Marshal session
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Write to file
	file := s.sessionFile(session.ID)
	if err := os.WriteFile(file, data, 0o644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// Load retrieves a session by ID
func (s *FileStore) Load(ctx context.Context, id string) (*Session, error) {
	file := s.sessionFile(id)

	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// List returns all sessions
func (s *FileStore) List(ctx context.Context, workspaceID string) ([]*Session, error) {
	// Ensure directory exists
	if err := os.MkdirAll(s.sessionDir(), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}

	// Read directory
	entries, err := os.ReadDir(s.sessionDir())
	if err != nil {
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var sessions []*Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "session-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Extract ID from filename
		id := strings.TrimPrefix(entry.Name(), "session-")
		id = strings.TrimSuffix(id, ".json")

		// Load session
		session, err := s.Load(ctx, id)
		if err != nil {
			continue // Skip invalid sessions
		}

		// Filter by workspace if specified
		if workspaceID != "" && session.WorkspaceID != workspaceID {
			continue
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// Remove deletes a session
func (s *FileStore) Remove(ctx context.Context, id string) error {
	// Remove metadata file
	metaFile := s.sessionFile(id)
	if err := os.Remove(metaFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove session file: %w", err)
	}

	// Remove log file
	logFile := s.logFile(id)
	if err := os.Remove(logFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove log file: %w", err)
	}

	return nil
}

// GetLogs retrieves logs for a session
func (s *FileStore) GetLogs(ctx context.Context, id string) (LogReader, error) {
	logFile := s.logFile(id)

	file, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("logs not found for session: %s", id)
		}
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &fileLogReader{file: file}, nil
}

// fileLogReader implements LogReader for file-based logs
type fileLogReader struct {
	file *os.File
}

func (r *fileLogReader) Read(p []byte) (n int, err error) {
	return r.file.Read(p)
}

func (r *fileLogReader) Close() error {
	return r.file.Close()
}

// SaveLogs saves logs for a session
func (s *FileStore) SaveLogs(ctx context.Context, id string, reader io.Reader) error {
	// Ensure directory exists
	if err := os.MkdirAll(s.sessionDir(), 0o755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	logFile := s.logFile(id)
	file, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to write logs: %w", err)
	}

	return nil
}
