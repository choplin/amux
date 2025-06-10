# 5. Command Structure with Subcommands

Date: 2025-06-10

## Status

Accepted

## Context

As the tool evolves to support both workspace management and agent multiplexing, we need a clear command structure
that prevents ambiguity and allows for future growth. Without proper organization, commands like `list` or `create`
are unclear about what they operate on.

## Decision

Establish a subcommand-based CLI structure with clear command groups.

## Command Structure

```bash
# Workspace commands
amux workspace create <name>    # alias: amux ws create
amux workspace list            # alias: amux ws list
amux workspace get <id>        # alias: amux ws get
amux workspace remove <id>     # alias: amux ws remove
amux workspace prune           # alias: amux ws prune

# Agent commands (future)
amux agent run <agent> [opts]  # alias: amux run
amux agent list               # alias: amux ps
amux agent attach <session>   # alias: amux attach
amux agent stop <session>     # no alias

# MCP server
amux mcp [options]            # no subcommand needed
```

## Rationale

**Why subcommands:**

- **No ambiguity**: `amux list` doesn't tell you what it lists
- **Extensibility**: Easy to add new command groups (config, plugin, etc.)
- **Discoverability**: `amux --help` shows clear command groups
- **Industry standard**: Follows patterns from docker, kubectl, gh

**Aliases for common operations:**

- `amux run` → `amux agent run` (primary feature gets short alias)
- `amux ps` → `amux agent list` (familiar from docker)
- `amux ws` → `amux workspace` (common commands get short form)

## Consequences

- Clear separation between workspace and agent operations
- Room for future feature additions without breaking changes
- Slightly more typing for explicit commands (mitigated by aliases)
- Consistent with user expectations from modern CLI tools
