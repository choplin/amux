# 022. Type-Safe Identifiers for Workspaces and Sessions

Date: 2025-06-15

## Status

Accepted

## Context

Amux uses various forms of identifiers for workspaces and sessions:

1. **Full UUID** - The complete unique identifier (e.g., `3b3509c3-4f5e-4b8a-9f3e-3f5e4b8a9f3e`)
2. **Index** - Short numeric identifier for convenience (e.g., `1`, `2`, `3`)
3. **Name** - Human-readable name (e.g., `fix-auth-bug`, `feat-logging`)

However, the codebase had several issues:

1. **Inconsistent API behavior**: Some methods accepted names (`GetByName`), others only IDs (`Get`)
2. **MCP tool inconsistency**: Some tools could resolve names while others couldn't
3. **Type unsafety**: All identifiers were passed as `string`, making it unclear what was accepted
4. **Code duplication**: Resolution logic was scattered across different methods

This led to a poor developer experience where users had to remember which commands accepted which identifier types.

## Decision

We will introduce dedicated types for identifiers to ensure type safety and consistent behavior:

### Type Hierarchy

For both workspaces and sessions:

```go
// workspace package
type ID string        // Full UUID
type Index string     // Short numeric identifier (1, 2, 3...)
type Name string      // Human-readable name
type Identifier string // Can be any of: ID, Index, or Name

// session package (same pattern)
type ID string
type Index string
type Name string
type Identifier string
```

### API Design

Each manager will have clear, type-safe methods:

```go
// Strict methods - only accept specific types
func (m *Manager) Get(id ID) (*Workspace, error)

// Flexible resolution - accepts any identifier type
func (m *Manager) ResolveWorkspace(identifier Identifier) (*Workspace, error)
```

The `ResolveWorkspace`/`ResolveSession` methods will contain all resolution logic:

1. Try as full ID
2. Try as index (short ID)
3. Try as name

### MCP Parameter Naming

To make the API more user-friendly, MCP tool parameters will be renamed:

- `workspace_id` → `workspace_identifier`
- `session_id` → `session_identifier`

With descriptions clarifying: "Workspace ID, index, or name"

## Consequences

### Positive

1. **Type safety**: Compile-time guarantees about what identifier types are accepted
2. **Consistent behavior**: All MCP tools and CLI commands accept any identifier type
3. **Better developer experience**: Clear from the type signature what's accepted
4. **Reduced code duplication**: Resolution logic centralized in one method
5. **Self-documenting code**: Types make the API contract explicit

### Negative

1. **More verbose**: Requires type conversions like `workspace.ID(someString)`
2. **Breaking change**: Internal APIs changed (though external behavior preserved)
3. **Learning curve**: Developers need to understand the type hierarchy

### Neutral

1. **Migration effort**: All callers need to be updated to use new types
2. **Testing updates**: Test code needs to use proper type conversions

## Implementation Notes

The implementation follows these principles:

1. **Backward compatibility**: External APIs (CLI, MCP) maintain the same behavior
2. **Fail-fast**: Type mismatches caught at compile time
3. **Single source of truth**: Resolution logic only in `Resolve*` methods
4. **Consistent patterns**: Same type hierarchy for both workspaces and sessions

## Examples

### Before (inconsistent and unclear)

```go
// Which methods accept names? Unclear from signatures
ws, err := manager.Get(identifier)        // ID only? Or name too?
ws, err := manager.GetByName(name)        // Redundant method
sess, err := sessionManager.GetSession(id) // What about names?
```

### After (type-safe and consistent)

```go
// Clear from types what's accepted
ws, err := manager.Get(workspace.ID("uuid..."))              // ID only
ws, err := manager.ResolveWorkspace(workspace.Identifier("1"))          // Index
ws, err := manager.ResolveWorkspace(workspace.Identifier("my-feature")) // Name
ws, err := manager.ResolveWorkspace(workspace.Identifier("uuid..."))    // ID

// Same pattern for sessions
sess, err := manager.Get(session.ID("uuid..."))
sess, err := manager.ResolveSession(session.Identifier("debug-session"))
```

## References

- Issue #138: MCP tool inconsistency
- PR #140: Implementation of type-safe identifiers
- Similar patterns in Kubernetes (ObjectMeta) and Docker (container references)
