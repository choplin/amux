// Package app provides dependency injection container for the application
package app

import (
	"fmt"

	"github.com/aki/amux/internal/core/agent"
	"github.com/aki/amux/internal/core/config"
	"github.com/aki/amux/internal/core/idmap"
	"github.com/aki/amux/internal/core/session"
	"github.com/aki/amux/internal/core/workspace"
)

// Container holds all manager instances and their dependencies
type Container struct {
	// ProjectRoot is the root directory of the amux project
	ProjectRoot string

	// Core managers
	ConfigManager    *config.Manager
	WorkspaceManager *workspace.Manager
	SessionManager   *session.Manager
	AgentManager     *agent.Manager

	// Shared dependencies
	IDMapper *idmap.IDMapper
}

// NewContainer creates a new container with all managers initialized in dependency order
func NewContainer(projectRoot string) (*Container, error) {
	c := &Container{
		ProjectRoot: projectRoot,
	}

	// Initialize config manager (no dependencies)
	c.ConfigManager = config.NewManager(projectRoot)

	// Check if amux is initialized
	if !c.ConfigManager.IsInitialized() {
		return nil, fmt.Errorf("amux not initialized in %s: run 'amux init' first", projectRoot)
	}

	// Initialize workspace manager (depends on config)
	var err error
	c.WorkspaceManager, err = workspace.NewManager(c.ConfigManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace manager: %w", err)
	}

	// Initialize shared ID mapper (used by multiple managers)
	c.IDMapper, err = idmap.NewIDMapper(c.ConfigManager.GetAmuxDir())
	if err != nil {
		return nil, fmt.Errorf("failed to create ID mapper: %w", err)
	}

	// Initialize agent manager (depends on config)
	c.AgentManager = agent.NewManager(c.ConfigManager)

	// Initialize session manager (depends on workspace, agent, and idmapper)
	c.SessionManager, err = session.NewManager(
		c.ConfigManager.GetAmuxDir(),
		c.WorkspaceManager,
		c.AgentManager,
		c.IDMapper,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}

	return c, nil
}

// NewContainerWithoutInit creates a container without checking initialization status.
// This is useful for commands that don't require an initialized amux project (e.g., init, version).
func NewContainerWithoutInit(projectRoot string) *Container {
	c := &Container{
		ProjectRoot: projectRoot,
	}

	// Only initialize config manager
	c.ConfigManager = config.NewManager(projectRoot)

	return c
}
