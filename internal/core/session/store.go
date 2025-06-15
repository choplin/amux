package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aki/amux/internal/filemanager"
)

// FileStore implements Store using YAML files with concurrent access safety
type FileStore struct {
	basePath string
	mu       sync.RWMutex // For in-process coordination
	fm       *filemanager.Manager[Info]
}

// NewFileStore creates a new file-based session store
func NewFileStore(basePath string) (*FileStore, error) {
	// Ensure sessions directory exists
	sessionsDir := filepath.Join(basePath, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	return &FileStore{
		basePath: sessionsDir,
		fm:       filemanager.NewManager[Info](),
	}, nil
}

// Save saves session info to a YAML file with file locking
func (s *FileStore) Save(info *Info) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.sessionPath(info.ID)
	return s.fm.Write(path, info)
}

// Load loads session info from a YAML file with shared lock
func (s *FileStore) Load(id string) (*Info, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.sessionPath(id)
	info, _, err := s.fm.Read(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSessionNotFound{ID: id}
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	return info, nil
}

// List lists all session infos
func (s *FileStore) List() ([]*Info, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessions []*Info
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		// Extract ID from filename (session-{id}.yaml)
		id := entry.Name()
		id = id[0 : len(id)-5] // Remove .yaml extension
		if len(id) > 8 && id[0:8] == "session-" {
			id = id[8:] // Remove session- prefix
		}

		info, err := s.Load(id)
		if err != nil {
			// Skip sessions that can't be loaded
			continue
		}

		sessions = append(sessions, info)
	}

	return sessions, nil
}

// Delete deletes a session info file with exclusive lock
func (s *FileStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.sessionPath(id)

	// First check if file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return ErrSessionNotFound{ID: id}
		}
		return fmt.Errorf("failed to check session file: %w", err)
	}

	// Delete the file with file locking
	if err := s.fm.Delete(path); err != nil {
		if os.IsNotExist(err) {
			// File was deleted between check and remove - still treat as not found
			return ErrSessionNotFound{ID: id}
		}
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	// Also remove the session storage directory if it exists
	sessionDir := filepath.Join(s.basePath, id)
	if err := os.RemoveAll(sessionDir); err != nil && !os.IsNotExist(err) {
		// Log error but don't fail the delete operation
		// The session is already deleted from a functional perspective
		fmt.Printf("Warning: failed to remove session storage directory: %v\n", err)
	}

	return nil
}

// Update safely updates a session info using CAS
func (s *FileStore) Update(id string, updateFunc func(info *Info) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.sessionPath(id)
	return s.fm.Update(path, func(info *Info) error {
		// Ensure ID is preserved
		info.ID = id
		return updateFunc(info)
	})
}

// sessionPath returns the file path for a session
func (s *FileStore) sessionPath(id string) string {
	return filepath.Join(s.basePath, fmt.Sprintf("session-%s.yaml", id))
}

// CreateSessionStorage creates a storage directory for a session and returns the path
func (s *FileStore) CreateSessionStorage(sessionID string) (string, error) {
	// Create storage directory under sessions/{sessionID}/storage
	storagePath := filepath.Join(s.basePath, sessionID, "storage")
	if err := os.MkdirAll(storagePath, 0o755); err != nil {
		return "", fmt.Errorf("failed to create session storage directory: %w", err)
	}
	return storagePath, nil
}
