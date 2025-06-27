// Package idmap provides ID mapping functionality for Amux entities.
package idmap

import (
	"fmt"

	"github.com/aki/amux/internal/core/index"
)

// WorkspaceID represents a workspace identifier
type WorkspaceID string

// SessionID represents a session identifier
type SessionID string

// Mapper is a generic ID mapper for type-safe ID management
type Mapper[T ~string] struct {
	entityType   index.EntityType
	indexManager index.Manager
}

// newMapper creates a new generic ID mapper
func newMapper[T ~string](amuxDir string, entityType index.EntityType) (*Mapper[T], error) {
	indexManager, err := index.NewManager(amuxDir)
	if err != nil {
		return nil, err
	}

	return &Mapper[T]{
		entityType:   entityType,
		indexManager: indexManager,
	}, nil
}

// NewWorkspaceIDMapper creates a new ID mapper for workspaces
func NewWorkspaceIDMapper(amuxDir string) (*Mapper[WorkspaceID], error) {
	return newMapper[WorkspaceID](amuxDir, index.EntityTypeWorkspace)
}

// NewSessionIDMapper creates a new ID mapper for sessions
func NewSessionIDMapper(amuxDir string) (*Mapper[SessionID], error) {
	return newMapper[SessionID](amuxDir, index.EntityTypeSession)
}

// Add adds an ID mapping and returns the index
func (m *Mapper[T]) Add(id T) (string, error) {
	idx, err := m.indexManager.Acquire(m.entityType, string(id))
	if err != nil {
		return "", err
	}
	return idx.String(), nil
}

// GetFull returns the full ID for an index
func (m *Mapper[T]) GetFull(indexStr string) (T, bool) {
	var idx index.Index
	if _, err := fmt.Sscanf(indexStr, "%d", &idx); err != nil {
		return T(""), false
	}
	fullID, found := m.indexManager.GetByIndex(m.entityType, idx)
	return T(fullID), found
}

// GetIndex returns the index for a full ID
func (m *Mapper[T]) GetIndex(id T) (string, bool) {
	idx, found := m.indexManager.Get(m.entityType, string(id))
	if !found {
		return "", false
	}
	return idx.String(), true
}

// Remove removes an ID mapping
func (m *Mapper[T]) Remove(id T) error {
	return m.indexManager.Release(m.entityType, string(id))
}

// Reconcile removes index entries for IDs that no longer exist
// Returns the number of orphaned entries that were cleaned up
func (m *Mapper[T]) Reconcile(existingIDs []T) (int, error) {
	// Convert to strings
	strIDs := make([]string, len(existingIDs))
	for i, id := range existingIDs {
		strIDs[i] = string(id)
	}
	return m.indexManager.Reconcile(m.entityType, strIDs)
}
