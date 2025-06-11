package common

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// IDMappings stores the mappings between short and full IDs
type IDMappings struct {
	WorkspaceCounter  int               `yaml:"workspace_counter"`
	SessionCounter    int               `yaml:"session_counter"`
	Workspaces        map[string]string `yaml:"workspaces"` // short -> full
	Sessions          map[string]string `yaml:"sessions"`   // short -> full
	ReverseWorkspaces map[string]string `yaml:"-"`          // full -> short (not persisted)
	ReverseSessions   map[string]string `yaml:"-"`          // full -> short (not persisted)
}

// IDMapper manages ID mappings with persistence
type IDMapper struct {
	mu       sync.RWMutex
	filePath string
	mappings *IDMappings
	wsGen    *ShortIDGenerator
	sessGen  *ShortIDGenerator
}

// NewIDMapper creates a new ID mapper
func NewIDMapper(amuxDir string) (*IDMapper, error) {
	filePath := filepath.Join(amuxDir, "id-mappings.yaml")
	mapper := &IDMapper{
		filePath: filePath,
		wsGen:    NewShortIDGenerator(),
		sessGen:  NewShortIDGenerator(),
	}

	if err := mapper.load(); err != nil {
		return nil, err
	}

	return mapper, nil
}

// load loads mappings from file or creates new ones
func (m *IDMapper) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize empty mappings
	m.mappings = &IDMappings{
		Workspaces:        make(map[string]string),
		Sessions:          make(map[string]string),
		ReverseWorkspaces: make(map[string]string),
		ReverseSessions:   make(map[string]string),
	}

	// Load from file if exists
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, that's OK
			return nil
		}
		return fmt.Errorf("failed to read ID mappings: %w", err)
	}

	if err := yaml.Unmarshal(data, m.mappings); err != nil {
		return fmt.Errorf("failed to parse ID mappings: %w", err)
	}

	// Set generators to correct counter values
	m.wsGen.SetCounter(m.mappings.WorkspaceCounter)
	m.sessGen.SetCounter(m.mappings.SessionCounter)

	// Build reverse mappings
	for short, full := range m.mappings.Workspaces {
		m.mappings.ReverseWorkspaces[full] = short
	}
	for short, full := range m.mappings.Sessions {
		m.mappings.ReverseSessions[full] = short
	}

	return nil
}

// save saves mappings to file
func (m *IDMapper) save() error {
	// Update counters before saving
	m.mappings.WorkspaceCounter = m.wsGen.GetCounter()
	m.mappings.SessionCounter = m.sessGen.GetCounter()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := yaml.Marshal(m.mappings)
	if err != nil {
		return fmt.Errorf("failed to marshal ID mappings: %w", err)
	}

	if err := os.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write ID mappings: %w", err)
	}

	return nil
}

// AddWorkspace adds a workspace ID mapping
func (m *IDMapper) AddWorkspace(fullID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already mapped
	if shortID, exists := m.mappings.ReverseWorkspaces[fullID]; exists {
		return shortID, nil
	}

	// Generate new short ID
	shortID := m.wsGen.Next()
	m.mappings.Workspaces[shortID] = fullID
	m.mappings.ReverseWorkspaces[fullID] = shortID

	return shortID, m.save()
}

// AddSession adds a session ID mapping
func (m *IDMapper) AddSession(fullID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already mapped
	if shortID, exists := m.mappings.ReverseSessions[fullID]; exists {
		return shortID, nil
	}

	// Generate new short ID
	shortID := m.sessGen.Next()
	m.mappings.Sessions[shortID] = fullID
	m.mappings.ReverseSessions[fullID] = shortID

	return shortID, m.save()
}

// GetWorkspaceFull returns the full ID for a short workspace ID
func (m *IDMapper) GetWorkspaceFull(shortID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fullID, exists := m.mappings.Workspaces[shortID]
	return fullID, exists
}

// GetSessionFull returns the full ID for a short session ID
func (m *IDMapper) GetSessionFull(shortID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fullID, exists := m.mappings.Sessions[shortID]
	return fullID, exists
}

// GetWorkspaceShort returns the short ID for a full workspace ID
func (m *IDMapper) GetWorkspaceShort(fullID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shortID, exists := m.mappings.ReverseWorkspaces[fullID]
	return shortID, exists
}

// GetSessionShort returns the short ID for a full session ID
func (m *IDMapper) GetSessionShort(fullID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shortID, exists := m.mappings.ReverseSessions[fullID]
	return shortID, exists
}

// RemoveWorkspace removes a workspace ID mapping
func (m *IDMapper) RemoveWorkspace(fullID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if shortID, exists := m.mappings.ReverseWorkspaces[fullID]; exists {
		delete(m.mappings.Workspaces, shortID)
		delete(m.mappings.ReverseWorkspaces, fullID)
		return m.save()
	}

	return nil
}

// RemoveSession removes a session ID mapping
func (m *IDMapper) RemoveSession(fullID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if shortID, exists := m.mappings.ReverseSessions[fullID]; exists {
		delete(m.mappings.Sessions, shortID)
		delete(m.mappings.ReverseSessions, fullID)
		return m.save()
	}

	return nil

}
