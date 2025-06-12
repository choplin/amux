# 10. MCP Bridge Tools

Date: 2025-01-11

## Status

Accepted

## Context

The Model Context Protocol (MCP) specification defines three core primitives:
- **Resources**: Read-only data access
- **Tools**: State-changing operations
- **Prompts**: Guided workflows

While amux has implemented all three primitives as designed in ADR-009, we discovered that many MCP clients
(including Claude Code) have limited or no support for reading resources directly. This creates a barrier
to adoption since clients cannot access essential workspace data.

The MCP ecosystem is still evolving, and client implementations vary significantly in their feature support:
- Most clients fully support tools
- Resource support is inconsistent or missing
- Prompt support is often limited

## Decision

We will implement "bridge" tools that provide tool-based access to MCP resources and prompts. These bridge
tools act as a compatibility layer, allowing clients without native resource support to access the same data
through the tools interface.

Bridge tools will:
1. Use a clear naming convention with prefixes (`resource_`, `prompt_`)
2. Return identical data to their resource/prompt counterparts
3. Share implementation logic with resources to ensure consistency
4. Be clearly documented as a compatibility layer

The following bridge tools will be implemented:
- `resource_workspace_list` - Bridge to `amux://workspace`
- `resource_workspace_get` - Bridge to `amux://workspace/{id}`
- `resource_workspace_browse` - Bridge to `amux://workspace/{id}/files`
- `prompt_list` - List available prompts
- `prompt_get` - Get specific prompt by name

## Consequences

### Positive

- **Immediate compatibility**: All MCP clients can access amux data regardless of their resource support
- **No client changes required**: Works with existing tool-only clients
- **Graceful degradation**: Clients can use native resources when available, bridge tools otherwise
- **Consistent data**: Shared implementation ensures resources and bridge tools return identical data
- **Easy migration path**: When clients add resource support, they can switch without breaking changes

### Negative

- **Conceptual blur**: Bridge tools violate the clean separation between read (resources) and write (tools)
- **Increased API surface**: More tools to maintain and document
- **Potential confusion**: Developers might not understand why there are two ways to access the same data
- **Technical debt**: Bridge tools should eventually be deprecated when client support improves

### Mitigation Strategies

1. **Clear documentation**: Explicitly mark bridge tools as a compatibility layer
2. **Consistent naming**: Use prefixes to distinguish bridge tools from core tools
3. **Shared implementation**: Extract common logic to avoid duplication and ensure consistency
4. **Future deprecation path**: Plan to phase out bridge tools as the ecosystem matures

## Implementation Notes

The implementation follows these principles:

1. **DRY (Don't Repeat Yourself)**: Resources and bridge tools share core logic
   ```go
   func (s *ServerV2) getWorkspaceList() ([]workspaceInfo, error) {
       // Shared implementation
   }
   ```

2. **Type safety**: Reuse the same data structures
   ```go
   type workspaceInfo struct {
       ID          string `json:"id"`
       Name        string `json:"name"`
       // ... same structure for both resource and tool
   }
   ```

3. **Clear documentation**: Each bridge tool describes its purpose
   ```go
   "List all workspaces (bridge to amux://workspace resource)"
   ```

## References

- [MCP Specification](https://modelcontextprotocol.io/docs/concepts/architecture)
- [ADR-009: MCP Resources and Prompts](./009-mcp-resources-and-prompts.md)
- [Issue #63: Implement bridge tools](https://github.com/choplin/amux/issues/63)
- [PR #65: Implementation](https://github.com/choplin/amux/pull/65)
