package index

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

// Manager handles index allocation and reuse
type Manager interface {
	// Acquire gets the next available index for an entity
	Acquire(entityType EntityType, entityID string) (Index, error)

	// Release marks an index as available for reuse
	Release(entityType EntityType, entityID string) error

	// Get retrieves the index for a given entity
	Get(entityType EntityType, entityID string) (Index, bool)

	// GetByIndex retrieves the entity ID for a given index
	GetByIndex(entityType EntityType, index Index) (string, bool)
}

// fileManager implements Manager with file-based persistence
type fileManager struct {
	mu        sync.RWMutex
	stateFile string
	state     *State
}

// NewManager creates a new index manager
func NewManager(amuxDir string) (Manager, error) {
	stateFile := filepath.Join(amuxDir, "index-state.yaml")
	m := &fileManager{
		stateFile: stateFile,
		state:     NewState(),
	}

	if err := m.load(); err != nil {
		return nil, fmt.Errorf("failed to load index state: %w", err)
	}

	return m, nil
}

// load loads the state from disk
func (m *fileManager) load() error {
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize empty state
			return m.save()
		}
		return err
	}

	return yaml.Unmarshal(data, m.state)
}

// save persists the state to disk
func (m *fileManager) save() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(m.stateFile), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(m.state)
	if err != nil {
		return err
	}

	return os.WriteFile(m.stateFile, data, 0644)
}

// Acquire gets the next available index
func (m *fileManager) Acquire(entityType EntityType, entityID string) (Index, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if entity already has an index
	if m.state.Active[entityType] == nil {
		m.state.Active[entityType] = make(map[int]string)
	}

	for idx, id := range m.state.Active[entityType] {
		if id == entityID {
			return Index(idx), nil
		}
	}

	// Try to reuse a released index
	if released := m.state.Released[entityType]; len(released) > 0 {
		// Sort to get the smallest available index
		sort.Ints(released)
		idx := released[0]
		m.state.Released[entityType] = released[1:]
		m.state.Active[entityType][idx] = entityID

		if err := m.save(); err != nil {
			return 0, err
		}

		return Index(idx), nil
	}

	// Allocate new index
	m.state.Counters[entityType]++
	idx := m.state.Counters[entityType]
	m.state.Active[entityType][idx] = entityID

	if err := m.save(); err != nil {
		return 0, err
	}

	return Index(idx), nil
}

// Release marks an index as available for reuse
func (m *fileManager) Release(entityType EntityType, entityID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state.Active[entityType] == nil {
		return nil // Nothing to release
	}

	// Find the index for this entity
	var indexToRelease int
	found := false
	for idx, id := range m.state.Active[entityType] {
		if id == entityID {
			indexToRelease = idx
			found = true
			break
		}
	}

	if !found {
		return nil // Entity doesn't have an index
	}

	// Remove from active and add to released
	delete(m.state.Active[entityType], indexToRelease)
	if m.state.Released[entityType] == nil {
		m.state.Released[entityType] = []int{}
	}
	m.state.Released[entityType] = append(m.state.Released[entityType], indexToRelease)

	return m.save()
}

// Get retrieves the index for a given entity
func (m *fileManager) Get(entityType EntityType, entityID string) (Index, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.state.Active[entityType] == nil {
		return 0, false
	}

	for idx, id := range m.state.Active[entityType] {
		if id == entityID {
			return Index(idx), true
		}
	}

	return 0, false
}

// GetByIndex retrieves the entity ID for a given index
func (m *fileManager) GetByIndex(entityType EntityType, index Index) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.state.Active[entityType] == nil {
		return "", false
	}

	entityID, exists := m.state.Active[entityType][int(index)]
	return entityID, exists
}
