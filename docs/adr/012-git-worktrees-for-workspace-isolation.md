# ADR-012: Git Worktrees for Workspace Isolation

Date: 2025-06-12

## Status

Accepted

## Context

Amux needs to provide isolated development environments where multiple AI agents can work simultaneously without
interfering with each other. We need a solution that:

- Provides true filesystem isolation between agents
- Allows parallel development on different features
- Integrates with existing git workflows
- Enables easy merging of changes back to main
- Is familiar to developers

Several options were considered:

1. **Shared working directory with branches** - Agents work in the same directory but on different branches
2. **Docker containers** - Each agent gets a containerized environment
3. **Virtual machines** - Full VM isolation per agent
4. **Git worktrees** - Separate working directories linked to the same repository
5. **Copy repositories** - Full clones for each workspace

## Decision

We will use Git worktrees to provide isolated workspaces for AI agents.

Each workspace will:

- Be created as a new git worktree
- Have its own dedicated branch
- Exist in a separate directory under `.amux/workspaces/`
- Maintain full filesystem isolation from other workspaces

## Consequences

### Positive

- **True filesystem isolation** - Agents can modify files without affecting others
- **Parallel development** - Multiple agents can work simultaneously
- **Standard git workflows** - Merging is done through normal git operations
- **Lightweight branching** - Worktrees share the same git object database
- **Developer familiarity** - Git worktrees are a standard git feature
- **Easy cleanup** - Removing a worktree is a simple operation

### Negative

- **Disk space usage** - Each workspace needs a full copy of the working tree
- **Worktree complexity** - Requires understanding of git worktree mechanics
- **Git version dependency** - Requires git 2.5+ for worktree support
- **Potential for orphaned worktrees** - Need cleanup mechanisms

### Neutral

- Worktrees are linked to the main repository
- Changes still need to be committed and pushed
- Branch management becomes more important

## Implementation Notes

1. Workspaces are created in `.amux/workspaces/workspace-{name}-{timestamp}-{hash}/`
2. Each workspace gets a branch named `amux/workspace-{name}-{timestamp}-{hash}`
3. Workspace metadata is stored both in the main repo and within each workspace
4. Cleanup involves removing both the worktree and the branch

## Alternatives Considered

### Shared Directory with Branches

**Pros**: Simple, no extra disk space
**Cons**: No isolation, agents interfere with each other, can't work on same files

### Docker Containers

**Pros**: Strong isolation, reproducible environments
**Cons**: Complexity, requires Docker, harder git integration, resource overhead

### Virtual Machines

**Pros**: Complete isolation
**Cons**: Heavy resource usage, complex setup, slow creation

### Full Repository Clones

**Pros**: Complete isolation, simple concept
**Cons**: Disk space for git objects, slower creation, complex remote management

## References

- [Git Worktree Documentation](https://git-scm.com/docs/git-worktree)
- [ADR-001: Initial Architecture Decisions](001-initial-architecture.md)
