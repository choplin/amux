# Command Reference

This page provides a comprehensive reference for all Amux commands.

## Command Structure

Amux uses a hierarchical command structure with aliases for common operations:

```text
amux <category> <action> [arguments] [flags]
```

Most commands have shorter aliases for convenience.

## Global Flags

These flags can be used with any command:

- `--help`, `-h` - Show help for the command
- `--verbose`, `-v` - Enable verbose output
- `--quiet`, `-q` - Suppress non-error output

## Workspace Commands

Manage isolated Git worktree environments.

### `amux workspace create` (alias: `amux ws create`)

Create a new workspace.

```bash
amux ws create <name> [flags]
```

**Flags:**

- `--description`, `-d` - Workspace description
- `--branch`, `-b` - Use existing branch instead of creating new one
- `--base-branch` - Base branch for new workspace (default: main branch)

**Examples:**

```bash
# Create workspace with new branch
amux ws create feature-auth --description "Implement authentication"

# Create workspace from existing branch
amux ws create bugfix-ui --branch fix/ui-crash
```

### `amux workspace list` (alias: `amux ws list`)

List all workspaces.

```bash
amux ws list [flags]
```

**Flags:**

- `--format`, `-f` - Output format: table (default), json, yaml
- `--sort` - Sort by: name, created, updated
- `--filter` - Filter expression

**Examples:**

```bash
# List in table format
amux ws list

# Output as JSON for scripting
amux ws list --format json

# Sort by creation date
amux ws list --sort created
```

### `amux workspace show` (alias: `amux ws show`)

Show detailed information about a workspace.

```bash
amux ws show <workspace-id-or-name>
```

**Examples:**

```bash
# Show by name
amux ws show feature-auth

# Show by numeric ID
amux ws show 1
```

### `amux workspace cd` (alias: `amux ws cd`)

Enter workspace directory in a subshell.

```bash
amux ws cd <workspace-id-or-name>
```

**Examples:**

```bash
# Enter workspace
amux ws cd feature-auth

# Exit with 'exit' command or Ctrl+D
```

### `amux workspace remove` (alias: `amux ws remove`)

Remove a workspace and its Git worktree.

```bash
amux ws remove <workspace-id-or-name> [flags]
```

**Flags:**

- `--force`, `-f` - Skip confirmation prompt

**Examples:**

```bash
# Remove with confirmation
amux ws remove old-feature

# Force remove
amux ws remove old-feature --force
```

### `amux workspace prune` (alias: `amux ws prune`)

Remove old workspaces.

```bash
amux ws prune [flags]
```

**Flags:**

- `--days`, `-d` - Remove workspaces older than N days (default: 30)
- `--dry-run` - Show what would be removed without removing

**Examples:**

```bash
# Remove workspaces older than 7 days
amux ws prune --days 7

# Preview what would be removed
amux ws prune --days 7 --dry-run
```

## Session Commands

Manage AI agent sessions (preview feature).

### `amux session run` (alias: `amux run`)

Start a new agent session.

```bash
amux run <agent-id> [flags]
```

**Flags:**

- `--workspace`, `-w` - Workspace to run in (creates if not exists)
- `--name`, `-n` - Session name
- `--detach`, `-d` - Start in background

**Examples:**

```bash
# Run Claude in specific workspace
amux run claude --workspace feature-auth

# Run with custom session name
amux run gpt --name "security-review"
```

### `amux session list` (alias: `amux ps`)

List running sessions.

```bash
amux ps [flags]
```

**Flags:**

- `--all`, `-a` - Show all sessions including stopped ones
- `--format`, `-f` - Output format: table, json, yaml

**Examples:**

```bash
# List running sessions
amux ps

# Show all sessions
amux ps --all
```

### `amux session attach` (alias: `amux attach`)

Attach to a running session.

```bash
amux attach <session-id>
```

**Examples:**

```bash
# Attach to session
amux attach sess-abc123

# Detach with Ctrl+B, D (tmux default)
```

### `amux session stop`

Stop a running session.

```bash
amux session stop <session-id>
```

### `amux session remove` (alias: `amux session rm`)

Remove a stopped session.

```bash
amux session rm <session-id>
```

### `amux session logs`

View session output.

```bash
amux session logs <session-id> [flags]
```

**Flags:**

- `--follow`, `-f` - Follow output (like tail -f)
- `--lines`, `-n` - Number of lines to show (default: 50)

**Examples:**

```bash
# View last 50 lines
amux session logs sess-abc123

# Follow logs in real-time
amux session logs -f sess-abc123
```

### `amux tail`

Alias for `amux session logs -f`.

```bash
amux tail <session-id>
```

## Agent Commands

Configure AI agents.

### `amux agent list`

List configured agents.

```bash
amux agent list
```

### `amux agent show`

Show agent configuration.

```bash
amux agent show <agent-id>
```

### `amux agent add`

Add a new agent configuration.

```bash
amux agent add <agent-id> [flags]
```

**Flags:**

- `--name`, `-n` - Display name
- `--command`, `-c` - Command to execute
- `--env`, `-e` - Environment variables

## Configuration Commands

Manage Amux configuration.

### `amux config show`

Display current configuration.

```bash
amux config show [flags]
```

**Flags:**

- `--format`, `-f` - Output format: yaml (default), json, pretty

**Examples:**

```bash
# Show as YAML
amux config show

# Show as JSON
amux config show --format json

# Human-friendly format
amux config show --format pretty
```

### `amux config edit`

Edit configuration in your editor.

```bash
amux config edit
```

Uses `$EDITOR` environment variable (defaults to `vi`).

## MCP Server

Start Model Context Protocol server.

### `amux mcp`

Start MCP server for AI agent integration.

```bash
amux mcp [flags]
```

**Flags:**

- `--transport`, `-t` - Transport type: stdio (default), https
- `--port`, `-p` - Port for HTTPS transport
- `--auth` - Authentication type: none, bearer
- `--token` - Auth token for bearer auth

**Examples:**

```bash
# Start with stdio (for Claude Code)
amux mcp

# Start HTTPS server
amux mcp --transport https --port 3000
```

## Utility Commands

### `amux init`

Initialize Amux in current directory.

```bash
amux init
```

Creates `.amux/` directory structure.

### `amux version`

Show version information.

```bash
amux version
```

### `amux status`

Show overall system status.

```bash
amux status
```

Displays:

- Active workspaces count
- Running sessions
- System health

## Output Formats

Many commands support multiple output formats via `--format`:

- `table` - Human-readable table (default)
- `json` - JSON for scripting
- `yaml` - YAML format
- `pretty` - Enhanced human-readable

## Environment Variables

- `AMUX_HOME` - Override `.amux` directory location
- `AMUX_EDITOR` - Override default editor
- `AMUX_LOG_LEVEL` - Set log level: debug, info, warn, error

## Exit Codes

- `0` - Success
- `1` - General error
- `2` - Command syntax error
- `3` - Resource not found
- `4` - Permission denied
- `5` - Already exists
