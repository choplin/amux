# 33. Dependency Management Simplification

Date: 2025-06-27

## Status

Accepted

## Context

The amux codebase had scattered dependency initialization across multiple locations:

1. **CLI commands** had helper functions (`GetWorkspaceManager()`, `GetSessionManager()`) that each created their own manager instances
2. **MCP server** created managers independently with its own initialization logic
3. **Test setup** code duplicated manager creation in various test helpers
4. **IDMapper duplication**: Both WorkspaceManager and SessionManager created separate IDMapper instances

This scattered approach led to:

- Code duplication across different parts of the codebase (~70% duplication in initialization code)
- Inconsistent initialization patterns
- Difficulty in testing (hard to inject test doubles)
- Potential for initialization order bugs
- No single source of truth for dependency relationships

Analysis showed that most CLI commands only need 1-2 managers, not all of them. Creating a central dependency container would force unnecessary transitive dependencies throughout the codebase.

## Decision

Implement package-local setup functions to centralize dependency initialization while maintaining clean separation:

1. Each core package provides its own `SetupManager()` function
2. These functions handle all internal dependency initialization
3. Dependencies are explicitly passed where needed
4. The public API consists of only two main entry points:
   - `workspace.SetupManager(projectRoot)`
   - `session.SetupManager(projectRoot)`
5. Remove all helper wrapper functions - commands call setup functions directly

Implementation details:

```go
// In workspace package
func SetupManager(projectRoot string) (*Manager, error)

// In session package
func SetupManager(projectRoot string) (*Manager, error)
func SetupManagerWithWorkspace(projectRoot string, workspaceManager *workspace.Manager) (*Manager, error)
```

## Consequences

### Positive

- **Reduced complexity**: Dependencies reduced by ~80% compared to Container
- **Go-idiomatic**: Each package is self-contained with explicit dependencies
- **Better encapsulation**: Implementation details (ConfigManager, AgentManager, IDMapper) are hidden
- **Fixes IDMapper sharing**: WorkspaceManager now accepts external IDMapper via `NewManagerWithIDMapper()`
- **Clearer dependencies**: Each command imports only what it needs
- **Simpler to understand**: No DI framework or pattern to learn
- **No unnecessary abstractions**: Removed helper wrappers that added no value

### Negative

- **Some duplication**: Setup logic is duplicated between workspace and session setup functions
- **Less flexibility**: Harder to swap implementations for testing (but Go prefers concrete types anyway)

### Neutral

- **Migration from Container**: Requires updating all code that used Container (completed)
- **Documentation**: New pattern documented in `docs/dependency-management.md`
