# 33. Dependency Container Pattern

Date: 2025-06-26

## Status

Accepted

## Context

The amux codebase had scattered dependency initialization across multiple locations:

1. CLI commands had helper functions (`GetWorkspaceManager()`, `GetSessionManager()`) that each created their own manager instances
2. The MCP server created managers independently with its own initialization logic
3. The session.Factory pattern handled some dependencies but not all
4. Test setup code duplicated manager creation in various test helpers
5. IDMapper was created separately by both workspace.Manager and session.Factory

This scattered approach led to:

- Code duplication across different parts of the codebase
- Inconsistent initialization patterns
- Difficulty in testing (hard to inject test doubles)
- Potential for initialization order bugs
- No single source of truth for dependency relationships

## Decision

We will implement a simple Container pattern to centralize dependency creation and wiring:

1. Create an `app.Container` struct that holds all manager instances
2. Provide a single `NewContainer(projectRoot)` function that creates all managers in the correct dependency order
3. Update all initialization code (CLI helpers, MCP server, tests) to use the Container
4. Keep the existing concrete struct approach (no interfaces required)
5. Maintain backward compatibility by not changing manager APIs

The Container structure:

```go
type Container struct {
    ProjectRoot      string
    ConfigManager    *config.Manager
    WorkspaceManager *workspace.Manager
    SessionManager   *session.Manager
    AgentManager     *agent.Manager
    IDMapper         *idmap.IDMapper
}
```

## Consequences

### Positive

- **Single source of truth**: All dependency relationships are defined in one place
- **Reduced duplication**: Eliminates redundant initialization code across CLI and MCP
- **Clear initialization order**: Dependencies are created in the correct order within NewContainer
- **Improved testability**: Easy to create test containers or swap specific managers
- **Simplified maintenance**: Adding new managers requires changes in only one place
- **Type safety**: Compile-time verification of dependencies
- **No external dependencies**: Uses only Go standard patterns

### Negative

- **Additional abstraction layer**: Developers need to understand the Container pattern
- **Potential for god object**: Container could grow large if many more managers are added
- **Not using interfaces**: Still using concrete types, which limits some testing scenarios

### Neutral

- **Migration effort**: Existing code needs to be updated to use Container (completed in this implementation)
- **Documentation**: New pattern needs to be documented for contributors
