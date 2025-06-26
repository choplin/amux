# Dependency Management Architecture

## Overview

Amux uses a simple, explicit dependency management pattern that provides a clean API surface while hiding implementation details. This document describes how dependencies are structured and initialized throughout the project.

## Core Principles

1. **Two Main Interfaces**: External code (CLI commands, MCP server) only interacts with two managers:
   - `WorkspaceManager` - All workspace-related operations
   - `SessionManager` - All session-related operations

2. **Hidden Implementation Details**: Other managers and dependencies are internal implementation details:
   - `ConfigManager` - Configuration management
   - `AgentManager` - Agent definitions and lookup
   - `IDMapper` - Numeric ID to full ID mapping
   - `GitOperations` - Git worktree operations
   - `TmuxAdapter` - Tmux integration

3. **Explicit Dependencies**: No dependency injection frameworks or containers - just explicit initialization in setup functions.

## Architecture

```text
┌─────────────────────────────────────────────────┐
│             External Code (CLI/MCP)             │
└────────────────┬────────────────┬───────────────┘
                 │                │
                 ▼                ▼
         SetupWorkspace    SetupSession
         Manager()         Manager()
                 │                │
┌────────────────┴────────────────┴───────────────┐
│            Internal Dependencies                │
│                                                 │
│  ConfigManager ──┬──► WorkspaceManager         │
│       │          │         │                   │
│       │          │         ▼                   │
│       │          │    IDMapper                 │
│       │          │         ▲                   │
│       │          │         │                   │
│       ▼          │         │                   │
│  AgentManager    └──► SessionManager           │
│                                                 │
└─────────────────────────────────────────────────┘
```

## Setup Functions

The project provides two setup functions that handle all dependency initialization:

```go
// SetupWorkspaceManager creates a workspace manager with all its dependencies
func SetupWorkspaceManager(projectRoot string) (*workspace.Manager, error)

// SetupSessionManager creates a session manager with all its dependencies
func SetupSessionManager(projectRoot string) (*session.Manager, error)
```

### Implementation Details

These setup functions:

1. Create a `ConfigManager` for the project
2. Verify the project is initialized
3. Create all required dependencies in the correct order
4. Return the configured manager

### Example Implementation

```go
func SetupSessionManager(projectRoot string) (*session.Manager, error) {
    // Create config manager
    configManager := config.NewManager(projectRoot)
    if !configManager.IsInitialized() {
        return nil, fmt.Errorf("not initialized: run 'amux init' first")
    }

    // Create workspace manager (dependency)
    wsManager, err := workspace.NewManager(configManager)
    if err != nil {
        return nil, fmt.Errorf("failed to create workspace manager: %w", err)
    }

    // Create other dependencies
    agentManager := agent.NewManager(configManager)
    idMapper, err := idmap.NewIDMapper(configManager.GetAmuxDir())
    if err != nil {
        return nil, fmt.Errorf("failed to create ID mapper: %w", err)
    }

    // Create session manager with all dependencies
    return session.NewManager(
        configManager.GetAmuxDir(),
        wsManager,
        agentManager,
        idMapper,
    )
}
```

## Usage Patterns

### CLI Commands

```go
func runCommand(cmd *cobra.Command, args []string) error {
    projectRoot, err := config.FindProjectRoot()
    if err != nil {
        return err
    }

    // Get the manager you need
    sessionManager, err := app.SetupSessionManager(projectRoot)
    if err != nil {
        return err
    }

    // Use it
    return sessionManager.CreateSession(ctx, opts)
}
```

### MCP Server

```go
func NewMCPServer(projectRoot string) (*Server, error) {
    wsManager, err := app.SetupWorkspaceManager(projectRoot)
    if err != nil {
        return nil, err
    }

    sessionManager, err := app.SetupSessionManager(projectRoot)
    if err != nil {
        return nil, err
    }

    return &Server{
        workspaceManager: wsManager,
        sessionManager:   sessionManager,
    }, nil
}
```

## Benefits

1. **Simple API**: Only two functions to understand
2. **No Circular Dependencies**: Each package depends only on what it needs
3. **Testability**: Easy to create test instances with mock dependencies
4. **Explicit**: Dependencies are clear and traceable
5. **Flexible**: Easy to add new dependencies without changing the API

## Guidelines

### When to Use Setup Functions

- Use `SetupWorkspaceManager()` when you only need workspace operations
- Use `SetupSessionManager()` when you need session operations (includes workspace access)
- Never access internal managers (ConfigManager, AgentManager, etc.) directly

### Adding New Dependencies

If you need to add a new internal dependency:

1. Add it to the relevant manager's struct
2. Initialize it in the setup function
3. Keep it private - don't expose it in the public API

### Testing

For tests, you can:

- Use the setup functions with a test repository
- Create managers directly with mock dependencies
- Use the test helpers in `internal/tests/helpers`

## Migration from Container Pattern

Previously, the project used a Container pattern that exposed all managers. The new approach:

1. Reduces the API surface to just two managers
2. Hides implementation details
3. Prevents unnecessary coupling
4. Makes dependencies explicit at each call site

To migrate:

- Replace `container.SessionManager` with `SetupSessionManager(projectRoot)`
- Replace `container.WorkspaceManager` with `SetupWorkspaceManager(projectRoot)`
- Remove direct access to other managers
