# 34. Setup Functions Over Container Pattern

Date: 2025-06-27

## Status

Accepted

## Context

ADR 033 introduced a Container pattern to centralize dependency initialization. While this reduced code duplication by ~70%, it created new problems:

1. **Transitive dependencies**: The Container made every helper depend on all managers, even unused ones
2. **Hidden complexity**: The Container pattern hid but didn't reduce the actual dependency complexity
3. **Not Go-idiomatic**: Container/DI patterns are uncommon in Go, preferring explicit initialization
4. **IDMapper duplication**: The Container created its own IDMapper while WorkspaceManager created another

Analysis showed that most CLI commands only need 1-2 managers, not all of them. The Container forced unnecessary dependencies throughout the codebase.

## Decision

Replace the Container pattern with package-local setup functions:

1. Each core package provides its own `SetupManager()` function
2. These functions handle all internal dependency initialization
3. Dependencies are explicitly passed where needed
4. The public API consists of only two main entry points:
   - `workspace.SetupManager(projectRoot)`
   - `session.SetupManager(projectRoot)`

Implementation details:

```go
// In workspace package
func SetupManager(projectRoot string) (*Manager, error)

// In session package
func SetupManager(projectRoot string) (*Manager, error)
func SetupManagerWithWorkspace(projectRoot string, workspaceManager *workspace.Manager) (*Manager, error)
```

This supersedes ADR 033's Container pattern.

## Consequences

### Positive

- **Reduced complexity**: Dependencies reduced by ~80% compared to Container
- **Go-idiomatic**: Each package is self-contained with explicit dependencies
- **Better encapsulation**: Implementation details (ConfigManager, AgentManager, IDMapper) are hidden
- **Fixes IDMapper sharing**: WorkspaceManager now accepts external IDMapper via `NewManagerWithIDMapper()`
- **Clearer dependencies**: Each command imports only what it needs
- **Simpler to understand**: No DI framework or pattern to learn

### Negative

- **Some duplication**: Setup logic is duplicated between workspace and session setup functions
- **Less flexibility**: Harder to swap implementations for testing (but Go prefers concrete types anyway)

### Neutral

- **Migration from Container**: Requires updating all code that used Container (completed)
- **Documentation**: New pattern documented in `docs/dependency-management.md`
