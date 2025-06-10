package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// FileStore implements SessionStore using YAML files
type FileStore struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileStore creates a new file-based session store
func NewFileStore(basePath string) (*FileStore, error) {
	// Ensure sessions directory exists
	sessionsDir := filepath.Join(basePath, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	return &FileStore{
		basePath: sessionsDir,
	}, nil
}

// Save saves session info to a YAML file
func (s *FileStore) Save(info *SessionInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := yaml.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal session info: %w", err)
	}

	path := s.sessionPath(info.ID)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// Load loads session info from a YAML file
func (s *FileStore) Load(id string) (*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.sessionPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrSessionNotFound{ID: id}
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var info SessionInfo
	if err := yaml.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session info: %w", err)
	}

	return &info, nil
}

// List lists all session infos
func (s *FileStore) List() ([]*SessionInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	var sessions []*SessionInfo
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

// Delete deletes a session info file
func (s *FileStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.sessionPath(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return ErrSessionNotFound{ID: id}
		}
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	return nil
}

// sessionPath returns the file path for a session
func (s *FileStore) sessionPath(id string) string {
	return filepath.Join(s.basePath, fmt.Sprintf("session-%s.yaml", id))

}
