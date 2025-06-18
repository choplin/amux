# ADR-026: Separate Storage Endpoints for Workspace and Session

**Status**: Accepted

## Context

Currently, the MCP storage tools (`storage_read`, `storage_write`, `storage_list`) use a unified approach where each tool accepts either `workspace_identifier` OR `session_identifier` parameters. This design was implemented as part of ADR-021's general storage feature without explicit documentation of the unified vs separate endpoint decision.

### Current Implementation Problems

1. **Parameter Confusion**: Each storage tool has both `workspace_identifier` and `session_identifier` parameters, but only one can be used at a time
2. **Validation Overhead**: Every storage operation must validate that exactly one identifier is provided
3. **Type Safety Issues**: JSON Schema cannot express "exactly one of" constraints effectively
4. **Poor Developer Experience**: Users see unnecessary parameters and must understand the mutual exclusivity

### Code Example of Current Approach

```go
// Current validation in every storage handler
if workspaceID == "" && sessionID == "" {
    return nil, fmt.Errorf("either workspace_identifier or session_identifier must be provided")
}
if workspaceID != "" && sessionID != "" {
    return nil, fmt.Errorf("only one of workspace_identifier or session_identifier should be provided")
}
```

## Decision

Replace the unified storage endpoints with distinct workspace and session storage tools:

- Remove `storage_read`, `storage_write`, `storage_list`
- Add `workspace_storage_read`, `workspace_storage_write`, `workspace_storage_list`
- Add `session_storage_read`, `session_storage_write`, `session_storage_list`

## Rationale

### Benefits of Separation

1. **Clear Interface**: Each tool has only the parameters it needs
2. **Type Safety**: Proper JSON Schema definitions without complex constraints
3. **No Validation Needed**: Parameter presence is guaranteed by the tool definition
4. **Better Discoverability**: Users can easily find the right tool for their context
5. **Future Extensibility**: Easy to add workspace-specific or session-specific features

### Implementation Simplicity

The implementation is straightforward:

- Create dedicated handlers for workspace and session storage
- Define separate parameter structs with only required fields
- No validation needed for mutual exclusivity

### Consistency with Other Tools

This aligns with our existing pattern where workspace and session operations have separate tools (e.g., `workspace_create` vs `session_run`).

## Consequences

### Positive

- Improved developer experience with clearer tool interfaces
- Reduced error cases and validation code
- Better alignment with MCP tool design principles
- Easier to document and understand

### Negative

- More tools in the MCP interface (6 instead of 3)
- Minor code duplication in tool registration

### Note

Since amux is in pre-release (v0.x), we can make breaking changes. The unified tools are removed immediately rather than deprecated.

## Implementation Notes

The new tools will maintain the same functionality and behavior as the current unified tools, just with cleaner interfaces:

```go
// New parameter structs
type WorkspaceStorageParams struct {
    WorkspaceID string `json:"workspace_identifier"`
    Path        string `json:"path"`
}

type SessionStorageParams struct {
    SessionID string `json:"session_identifier"`
    Path      string `json:"path"`
}
```

## References

- ADR-021: Deprecate Mailbox in Favor of General Storage
- ADR-024: MCP Tool Discoverability
