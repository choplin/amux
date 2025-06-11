# 6. Short-hand Indices for Workspaces and Sessions

Date: 2025-06-11

## Status

Accepted

## Context

Users need to interact with workspaces and sessions frequently through the CLI. The current full IDs are long and
cumbersome to type:

- Workspace ID: `workspace-test-1234567890-abcd1234`
- Session ID: `session-agent-test-1234567890-efgh5678`

This creates friction in the user experience, especially for commands that are run frequently like:

- `amux ws get <id>`
- `amux agent attach <id>`
- `amux agent stop <id>`

## Decision

We will implement a short-hand index system that provides sequential numeric indices (1, 2, 3...) for convenient
reference to workspaces and sessions.

### Key Properties of Indices

1. **Sequential Generation**: Indices are assigned sequentially starting from 1
2. **Convenient Reference**: Used for quick typing in CLI commands
3. **Volatile**: Can be reused after an entity is deleted
4. **Stable During Lifetime**: Once assigned, an index remains unchanged while the entity exists
5. **Not True IDs**: These are indices for convenience, not permanent identifiers

### Implementation Details

- Indices are managed by a dedicated `IndexManager` in the `internal/core/index` package
- The manager handles index allocation, reuse of released indices, and persistence
- State is stored in `.amux/index/state.yaml` with support for multiple entity types
- Commands accept both indices and full IDs for backwards compatibility
- CLI output displays indices when available, falling back to full IDs

### Architecture

The index system uses a clean, extensible architecture:

```text
internal/core/index/
├── manager.go      # IndexManager interface & implementation
├── types.go        # Core types (Index, EntityType, State)
└── manager_test.go # Comprehensive test coverage
```

The `IDMapper` in `internal/core/common` provides a thin wrapper around `IndexManager` for backward compatibility.

### Naming Rationale

We explicitly chose the term "Index" over alternatives like "Short ID" because:

- "ID" implies permanence and uniqueness, which these values don't guarantee
- "Index" clearly indicates these are positional references that may be reused
- This naming sets correct expectations about the volatile nature of these values

## Consequences

### Positive

- Significantly improved user experience with shorter commands
- Maintains full backwards compatibility
- Clear naming prevents misunderstanding about the nature of these values
- Simple implementation with minimal overhead

### Negative

- Indices can be reused, which might cause confusion if users expect permanence
- Additional mapping file to maintain
- Slight complexity in ID resolution logic

### Neutral

- Users need to understand the difference between indices and IDs
- Documentation must clearly explain the volatile nature of indices
