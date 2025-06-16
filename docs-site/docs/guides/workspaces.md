---
sidebar_position: 1
---

# Workspace Management

Complete guide to managing Amux workspaces.

## Understanding Workspaces

Each Amux workspace is:

- A Git worktree (not a clone)
- An isolated branch
- A separate working directory
- Independent from other workspaces

## Creating Workspaces

### Basic Creation

```bash
amux ws create feature-name
```

### With Description

```bash
amux ws create feature-auth --description "Implement OAuth2 authentication"
```

### From Existing Branch

```bash
amux ws create bugfix --branch origin/fix/critical-bug
```

### With Custom Base Branch

```bash
amux ws create feature --base-branch develop
```

## Managing Workspaces

### List All Workspaces

```bash
amux ws list
amux ws list --format json  # For scripting
```

### Show Workspace Details

```bash
amux ws show feature-auth
amux ws show 1  # By ID
```

### Enter Workspace

```bash
amux ws cd feature-auth
# You're now in a subshell
# Exit with 'exit' or Ctrl+D
```

## Workspace Lifecycle

### Active Development

1. Create workspace
2. Make changes
3. Commit regularly
4. Push to remote when ready

### Cleanup

```bash
# Remove single workspace
amux ws remove old-feature

# Remove workspaces older than N days
amux ws prune --days 7

# Preview what would be removed
amux ws prune --days 7 --dry-run
```

## Workspace Storage

Each workspace has a dedicated storage directory:

```text
.amux/workspaces/{workspace-id}/storage/
```

Storage can be used for:

- Documentation and notes
- Configuration files
- Task-specific data
- Any files needed by AI agents

Access via MCP tools:

- `storage_write` - Write files to storage
- `storage_read` - Read files from storage
- `storage_list` - List storage contents

## Troubleshooting

### Workspace Not Found

- Check with `amux ws list`
- Ensure you're in an initialized project
- Try using the numeric ID instead of name

### Cannot Remove Workspace

- Exit the workspace first
- Check for uncommitted changes
- Use `--force` flag if necessary

### Workspace Corruption

If a workspace becomes corrupted:

```bash
# Remove forcefully
amux ws remove workspace-name --force

# Recreate if needed
amux ws create workspace-name
```
