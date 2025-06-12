# ADR-016: Workspace Directory Reorganization

Date: 2025-06-12

## Status

Accepted

## Context

The initial workspace directory structure placed all workspace-related files within the git worktree:

```text
.amux/workspaces/{workspace-id}/
└── worktree/
    ├── .amux/
    │   ├── context/
    │   │   ├── WORKING.md
    │   │   └── CLAUDE.workspace.md
    │   └── workspace.yaml
    └── ... (project files)
```

This structure had several issues:

1. **Git pollution**: The `.amux/` directory appeared in git status, requiring gitignore entries
2. **Workspace metadata mixing**: Workspace metadata lived inside the worktree it described
3. **Clean worktree principle violation**: Worktrees should be identical to regular checkouts
4. **Context file confusion**: Multiple potential locations for workspace context

Additionally, after implementing workspace context support (#77), we needed a standardized location
for context files that was discoverable but didn't pollute the git worktree.

## Decision

We will reorganize the workspace directory structure to separate metadata from the git worktree:

```text
.amux/workspaces/{workspace-id}/
├── workspace.yaml     # Workspace metadata
├── context.md        # Workspace-specific context (optional)
└── worktree/         # Clean git worktree
    └── ... (project files only)
```

Key changes:

1. **Move workspace.yaml** out of the worktree to the parent directory
2. **Establish context.md location** at the workspace root (not in worktree)
3. **Remove .amux/context/** directory structure entirely
4. **Keep worktrees clean** - no amux-specific files inside

The context file path is stored in the Workspace struct and exposed through:

- CLI: `amux ws show` displays the context path
- MCP: Workspace resources include `contextPath` field

## Consequences

### Positive

- **Clean git worktrees**: No amux files in git status
- **Clear separation**: Metadata about the workspace vs. workspace contents
- **Simpler structure**: One location for workspace metadata
- **Better discoverability**: Context file path available through standard interfaces
- **No gitignore needed**: Amux files stay outside the repository

### Negative

- **Breaking change**: Existing workspaces need migration
- **Tool updates**: All tools reading workspace data need updates
- **Documentation updates**: Examples and guides need revision

### Neutral

- **Context file remains optional**: Created only when needed
- **Backward compatibility**: Can detect and migrate old structures

## Implementation

1. Update workspace creation to use new structure
2. Update workspace loading to look in new location
3. Add migration logic for existing workspaces
4. Update all tools and documentation
5. Add `ContextPath` field to Workspace struct (#77)

## References

- Issue #76: Deprecate .amux/context folder in workspaces
- Issue #77: Support workspace context file
- PR #78: Reorganize workspace directory structure
- PR #79: Add workspace context file support
- ADR-009: MCP Resources and Prompts (workspace structure)
- ADR-011: Documentation Structure Strategy
