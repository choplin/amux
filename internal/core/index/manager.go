// Package index provides short index management for entity identification.
package index

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/gofrs/flock"
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
	mu        sync.Mutex   // For goroutine synchronization within process
	flock     *flock.Flock // For process synchronization across processes
	stateFile string
}

// NewManager creates a new index manager
func NewManager(amuxDir string) (Manager, error) {
	indexDir := filepath.Join(amuxDir, "index")
	stateFile := filepath.Join(indexDir, "state.yaml")
	lockFile := stateFile + ".lock"

	// Ensure directory exists
	if err := os.MkdirAll(indexDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create index directory: %w", err)
	}

	m := &fileManager{
		stateFile: stateFile,
		flock:     flock.New(lockFile),
	}

	return m, nil
}

// loadState loads the state from disk (must be called with lock held)
func (m *fileManager) loadState() (*State, error) {
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty state
			return NewState(), nil
		}
		return nil, err
	}

	state := NewState()
	if err := yaml.Unmarshal(data, state); err != nil {
		return nil, err
	}
	return state, nil
}

// saveState persists the state to disk (must be called with lock held)
func (m *fileManager) saveState(state *State) error {
	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(m.stateFile, data, 0644)
}

// Acquire gets the next available index
func (m *fileManager) Acquire(entityType EntityType, entityID string) (Index, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Acquire file lock
	if err := m.flock.Lock(); err != nil {
		return 0, fmt.Errorf("failed to acquire file lock: %w", err)
	}
	defer func() {
		_ = m.flock.Unlock()
	}()

	// Load current state
	state, err := m.loadState()
	if err != nil {
		return 0, err
	}

	// Check if entity already has an index
	if state.Active[entityType] == nil {
		state.Active[entityType] = make(map[int]string)
	}

	for idx, id := range state.Active[entityType] {
		if id == entityID {
			return Index(idx), nil
		}
	}

	// Try to reuse a released index
	if released := state.Released[entityType]; len(released) > 0 {
		// Sort to get the smallest available index
		sort.Ints(released)
		idx := released[0]
		state.Released[entityType] = released[1:]
		state.Active[entityType][idx] = entityID

		if err := m.saveState(state); err != nil {
			return 0, err
		}

		return Index(idx), nil
	}

	// Allocate new index
	state.Counters[entityType]++
	idx := state.Counters[entityType]
	state.Active[entityType][idx] = entityID

	if err := m.saveState(state); err != nil {
		return 0, err
	}

	return Index(idx), nil
}

// Release marks an index as available for reuse
func (m *fileManager) Release(entityType EntityType, entityID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Acquire file lock
	if err := m.flock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}
	defer func() {
		_ = m.flock.Unlock()
	}()

	// Load current state
	state, err := m.loadState()
	if err != nil {
		return err
	}

	if state.Active[entityType] == nil {
		return nil // Nothing to release
	}

	// Find the index for this entity
	var indexToRelease int
	found := false
	for idx, id := range state.Active[entityType] {
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
	delete(state.Active[entityType], indexToRelease)
	if state.Released[entityType] == nil {
		state.Released[entityType] = []int{}
	}
	state.Released[entityType] = append(state.Released[entityType], indexToRelease)

	return m.saveState(state)
}

// Get retrieves the index for a given entity
func (m *fileManager) Get(entityType EntityType, entityID string) (Index, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Acquire file lock for reading
	if err := m.flock.RLock(); err != nil {
		return 0, false
	}
	defer func() {
		_ = m.flock.Unlock()
	}()

	// Load current state
	state, err := m.loadState()
	if err != nil {
		return 0, false
	}

	if state.Active[entityType] == nil {
		return 0, false
	}

	for idx, id := range state.Active[entityType] {
		if id == entityID {
			return Index(idx), true
		}
	}

	return 0, false
}

// GetByIndex retrieves the entity ID for a given index
func (m *fileManager) GetByIndex(entityType EntityType, index Index) (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Acquire file lock for reading
	if err := m.flock.RLock(); err != nil {
		return "", false
	}
	defer func() {
		_ = m.flock.Unlock()
	}()

	// Load current state
	state, err := m.loadState()
	if err != nil {
		return "", false
	}

	if state.Active[entityType] == nil {
		return "", false
	}

	entityID, exists := state.Active[entityType][int(index)]
	return entityID, exists
}
