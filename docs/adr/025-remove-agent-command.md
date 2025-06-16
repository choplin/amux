# 25. Remove Agent Command

Date: 2025-06-17

## Status

Accepted

## Context

The `amux agent` command was created to provide a dedicated interface for viewing agent configurations. However, after implementation and usage, we identified significant overlap with the `amux config` command:

1. **Redundancy**: Agent configurations are already displayed by `amux config show`
2. **Confusion**: Users had two different commands to view essentially the same information
3. **Maintenance burden**: Two separate code paths for displaying agent configuration
4. **Inconsistent UX**: The agent command was read-only but lived at the same level as other CRUD commands

The `amux agent` command provided:

- `agent list` - List configured agents in table format
- `agent show <id>` - Show specific agent details

The `amux config` command already provides:

- `config show` - Shows all configuration including agents
- `config edit` - Edit configuration including agents
- `config validate` - Validate configuration

## Decision

Remove the `amux agent` command entirely and consolidate all configuration viewing and editing under the `amux config` command. This creates a single, clear path for all configuration-related tasks.

Changes made:

1. Removed `/internal/cli/commands/agent/` directory and all its files
2. Removed agent command registration from `root.go`
3. Updated references from `amux agent attach` to `amux session attach`
4. Updated documentation to remove agent command references

## Consequences

**Positive:**

- Simpler CLI interface with less cognitive overhead
- Single source of truth for configuration management
- Reduced code maintenance burden
- Clearer mental model: "config" for configuration, "session" for runtime

**Negative:**

- Breaking change for users who may have scripts using `amux agent`
- Loss of specialized table formatting for agent listings (can be added to config show later)

**Future considerations:**

- Could enhance `amux config show` with filtering options like `--agents-only` or `--agent <id>`
- Could add better formatting options to `config show` for specific sections
