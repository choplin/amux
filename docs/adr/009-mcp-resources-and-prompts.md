# 9. MCP Resources and Prompts Architecture

Date: 2025-06-12

## Status

Proposed

## Context

Amux currently implements only MCP Tools for all operations, including read-only operations like `workspace_info`.
This violates MCP design principles where:

- **Resources** should be used for read-only data access
- **Tools** should be used for state-changing operations
- **Prompts** should guide users through complex workflows

Current problems:

1. Tools are misused for reading data (`workspace_info`)
2. No discoverable way for AI agents to learn amux conventions
3. No guided workflows for common tasks
4. AI agents must figure out amux patterns through trial and error

## Decision

We will implement a proper MCP architecture with clear separation of concerns:

### 1. Resources (Read-only Data)

Implement resources for browsing and reading amux data:

```text
amux://workspace                        # List all workspaces
amux://workspace/{id}                   # Get workspace details with paths
amux://workspace/{id}/files[/{path}]   # Browse workspace files
amux://workspace/{id}/context           # Read workspace context
```

### 2. Tools (State Changes Only)

Keep tools strictly for operations that modify state:

```typescript
workspace_create(name, baseBranch, description?, agentId?)
workspace_remove(workspace_id)
// Future: session_start, session_stop enhancements
```

### 3. Prompts (Guided Workflows)

Implement prompts that teach AI agents how to use amux effectively:

- `start-issue-work` - Complete workflow for starting GitHub issue work
- `prepare-pr-submission` - Guide for preparing and submitting PRs
- `manage-multiple-tasks` - Best practices for concurrent development
- `ai-agent-collaboration` - Patterns for multi-agent workflows

### 4. Conventions Resource

The `amux://conventions` resource will return:

```json
{
  "paths": {
    "workspace_root": ".amux/workspaces/{workspace-id}/worktree/",
    "workspace_context": ".amux/workspaces/{workspace-id}/context.md",
    "session_storage": ".amux/sessions/{session-id}/storage/"
  },
  "patterns": {
    "branch_name": "amux/workspace-{name}-{timestamp}-{hash}",
    "workspace_id": "workspace-{name}-{timestamp}-{hash}"
  }
}
```

## Consequences

### Positive

1. **Better Discoverability**: AI agents can explore available resources and understand amux structure
2. **Guided Experience**: Prompts provide step-by-step workflows for common tasks
3. **Proper Separation**: Clear distinction between reading and writing operations
4. **Self-Documenting**: Conventions resource explains how amux works
5. **MCP Compliant**: Follows MCP design principles correctly

### Negative

1. **Breaking Change**: Clients using `workspace_info` tool will need updates
2. **More Complexity**: Three different MCP primitives instead of just tools
3. **Migration Effort**: Existing integrations need to adapt

### Neutral

1. Resources are cached differently than tool responses
2. Prompts require user interaction and confirmation
3. More granular permission model possible in future

## Implementation Plan

### Phase 1: Core Resources

1. Implement resource handler infrastructure
2. Add conventions resource
3. Add workspace listing and detail resources
4. Add file browsing resources

### Phase 2: Prompts

1. Implement prompt infrastructure
2. Add start-issue-work prompt with emphasis on understanding requirements
3. Add other workflow prompts

### Phase 3: Tool Cleanup

1. Mark `workspace_info` as deprecated
2. Remove after grace period
3. Update documentation

## Alternatives Considered

### 1. Keep Everything as Tools

- Simpler but violates MCP principles
- No guided workflows
- Poor discoverability

### 2. Resources Only for Conventions

- Minimal change but doesn't solve file browsing
- Still mixing concerns with tools

### 3. Complex Resource Hierarchy

- Considered resources for git operations
- Decided git CLI is better used directly
- Keep amux focused on workspace management

## Updates

### 2025-06-12: Removed Conventions Resource

After implementation review, we removed the `amux://conventions` resource because:

- Conventions (paths, naming patterns) are implementation details
- AI agents only need actual paths, not patterns to construct them
- Each workspace now includes its actual paths in the detail response
- This simplifies the API and makes it more practical

Workspace details now include:

- `paths.worktree` - Actual path to the git worktree
- `paths.context` - Path to context.md file
- `resources.*` - URIs to related resources

## References

- [MCP Resources Documentation](https://modelcontextprotocol.io/docs/concepts/resources)
- [MCP Prompts Documentation](https://modelcontextprotocol.io/docs/concepts/prompts)
- [MCP Tools Documentation](https://modelcontextprotocol.io/docs/concepts/tools)
- Issue #44: Implement MCP Resources and Prompts architecture
