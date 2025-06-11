package common

import (
	"fmt"

	"github.com/aki/amux/internal/core/index"
)

// IDMapper manages ID mappings using IndexManager
type IDMapper struct {
	indexManager index.Manager
}

// NewIDMapper creates a new ID mapper
func NewIDMapper(amuxDir string) (*IDMapper, error) {
	// Create index manager
	indexManager, err := index.NewManager(amuxDir)
	if err != nil {
		return nil, err
	}

	return &IDMapper{
		indexManager: indexManager,
	}, nil
}

// AddWorkspace adds a workspace ID mapping
func (m *IDMapper) AddWorkspace(fullID string) (string, error) {
	idx, err := m.indexManager.Acquire(index.EntityTypeWorkspace, fullID)
	if err != nil {
		return "", err
	}
	return idx.String(), nil
}

// AddSession adds a session ID mapping
func (m *IDMapper) AddSession(fullID string) (string, error) {
	idx, err := m.indexManager.Acquire(index.EntityTypeSession, fullID)
	if err != nil {
		return "", err
	}
	return idx.String(), nil
}

// GetWorkspaceFull returns the full ID for a workspace index
func (m *IDMapper) GetWorkspaceFull(indexStr string) (string, bool) {
	var idx index.Index
	if _, err := fmt.Sscanf(indexStr, "%d", &idx); err != nil {
		return "", false
	}
	return m.indexManager.GetByIndex(index.EntityTypeWorkspace, idx)
}

// GetSessionFull returns the full ID for a session index
func (m *IDMapper) GetSessionFull(indexStr string) (string, bool) {
	var idx index.Index
	if _, err := fmt.Sscanf(indexStr, "%d", &idx); err != nil {
		return "", false
	}
	return m.indexManager.GetByIndex(index.EntityTypeSession, idx)
}

// GetWorkspaceIndex returns the index for a full workspace ID
func (m *IDMapper) GetWorkspaceIndex(fullID string) (string, bool) {
	idx, found := m.indexManager.Get(index.EntityTypeWorkspace, fullID)
	if !found {
		return "", false
	}
	return idx.String(), true
}

// GetSessionIndex returns the index for a full session ID
func (m *IDMapper) GetSessionIndex(fullID string) (string, bool) {
	idx, found := m.indexManager.Get(index.EntityTypeSession, fullID)
	if !found {
		return "", false
	}
	return idx.String(), true
}

// RemoveWorkspace removes a workspace ID mapping
func (m *IDMapper) RemoveWorkspace(fullID string) error {
	return m.indexManager.Release(index.EntityTypeWorkspace, fullID)
}

// RemoveSession removes a session ID mapping
func (m *IDMapper) RemoveSession(fullID string) error {
	return m.indexManager.Release(index.EntityTypeSession, fullID)

}
