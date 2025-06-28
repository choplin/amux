package session

import (
	"fmt"

	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/workspace"
)

// Factory provides methods for creating session managers with dependencies
type Factory struct{}

// NewFactory creates a new session manager factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateManager creates a new session manager with all dependencies
func (f *Factory) CreateManager(configManager *config.Manager, workspaceManager *workspace.Manager) (*Manager, error) {
	// Create session ID mapper
	idMapper, err := idmap.NewSessionIDMapper(configManager.GetAmuxDir())
	if err != nil {
		return nil, fmt.Errorf("failed to create session ID mapper: %w", err)
	}

	// Create session manager
	manager, err := NewManager(configManager.GetAmuxDir(), workspaceManager, configManager, idMapper)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	return manager, nil
}

// CreateManagerWithDependencies creates a new session manager with provided dependencies
// This is useful for testing or when you already have the dependencies created
func (f *Factory) CreateManagerWithDependencies(
	basePath string,
	workspaceManager *workspace.Manager,
	configManager *config.Manager,
	idMapper *idmap.Mapper[idmap.SessionID],
) (*Manager, error) {
	return NewManager(basePath, workspaceManager, configManager, idMapper)
}
