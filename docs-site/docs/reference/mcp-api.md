---
sidebar_position: 3
---

# MCP API Reference

Complete reference for Amux Model Context Protocol implementation.

## Starting the MCP Server

```bash
# STDIO transport (for Claude Code)
amux mcp --git-root /path/to/project

# HTTP transport (experimental)
amux mcp --transport http --port 3000
```

## Complete CLI-MCP Mapping

### Workspace Operations

| CLI Command | MCP Tool/Resource | Parameters |
|-------------|-------------------|------------|
| `amux ws create <name>` | `workspace_create` | `name`, `description?`, `branch?`, `baseBranch?` |
| `amux ws list` | `resource_workspace_list` | - |
| `amux ws show <id>` | `resource_workspace_show` | `workspace_identifier` |
| `amux ws remove <id>` | `workspace_remove` | `workspace_identifier` |
| `amux ws cd <id>` | N/A (CLI only) | - |
| `amux ws prune` | N/A (CLI only) | - |
| N/A | `resource_workspace_browse` (disabled) | `workspace_identifier`, `path?` |

### Session Operations

| CLI Command | MCP Tool/Resource | Parameters |
|-------------|-------------------|------------|
| `amux run <agent>` | `session_run` | `agent_id`, `workspace_identifier`, `name?`, `description?`, `command?` |
| `amux ps` / `amux session list` | `resource_session_list` | - |
| `amux session show <id>` | `resource_session_show` | `session_identifier` |
| `amux session stop <id>` | `session_stop` | `session_identifier` |
| `amux session send-input <id>` | `session_send_input` | `session_identifier`, `input` |
| `amux attach <id>` | N/A (CLI only) | - |
| `amux session logs <id>` | `resource_session_output` | `session_identifier` |
| `amux tail <id>` | N/A (CLI only) | - |

### Storage Operations (MCP Only)

| CLI Command | MCP Tool | Parameters |
|-------------|----------|------------|
| N/A | `storage_read` | `workspace_identifier?`, `session_identifier?`, `path` |
| N/A | `storage_write` | `workspace_identifier?`, `session_identifier?`, `path`, `content` |
| N/A | `storage_list` | `workspace_identifier?`, `session_identifier?`, `path?` |

### Configuration Operations

| CLI Command | MCP Tool/Resource | Parameters |
|-------------|-------------------|------------|
| `amux config show` | N/A (CLI only) | - |
| `amux config edit` | N/A (CLI only) | - |
| `amux config validate` | N/A (CLI only) | - |

### Prompt Operations (MCP Only)

| CLI Command | MCP Tool | Parameters |
|-------------|----------|------------|
| N/A | `prompt_list` | - |
| N/A | `prompt_get` | `name` |

### Other CLI-Only Commands

| CLI Command | Purpose |
|-------------|---------|
| `amux init` | Initialize Amux in project |
| `amux hooks` | Manage lifecycle hooks |
| `amux workspace context` | Manage context files |
| `amux completion` | Generate shell completion |
| `amux version` | Show version info |

## MCP Tools Reference

### Workspace Management Tools

#### workspace_create

Create a new isolated git worktree-based workspace.

```typescript
workspace_create({
  name: string,              // Required: workspace name
  description?: string,      // Optional: workspace description
  branch?: string,          // Optional: use existing branch
  baseBranch?: string       // Optional: base branch for new workspace
})
```

#### workspace_remove

Remove a workspace and its associated git worktree.

```typescript
workspace_remove({
  workspace_identifier: string  // Workspace ID, index, or name
})
```

### Session Management Tools

#### session_run

Run an AI agent session in a workspace.

```typescript
session_run({
  agent_id: string,                // Required: agent to run
  workspace_identifier: string,    // Required: target workspace
  name?: string,                   // Optional: session name
  description?: string,            // Optional: session description
  command?: string,                // Optional: override command
  environment?: {[key: string]: string}  // Optional: env variables
})
```

#### session_stop

Stop a running agent session gracefully.

```typescript
session_stop({
  session_identifier: string  // Session ID, index, or name
})
```

#### session_send_input

Send input text to a running session.

```typescript
session_send_input({
  session_identifier: string,  // Session ID, index, or name
  input: string               // Text to send
})
```

### Storage Tools

#### storage_read

Read a file from workspace or session storage.

```typescript
storage_read({
  workspace_identifier?: string,  // Either workspace...
  session_identifier?: string,    // ...or session (one required)
  path: string                   // Relative path in storage
})
```

#### storage_write

Write a file to workspace or session storage.

```typescript
storage_write({
  workspace_identifier?: string,  // Either workspace...
  session_identifier?: string,    // ...or session (one required)
  path: string,                  // Relative path in storage
  content: string                // File content
})
```

#### storage_list

List files in workspace or session storage.

```typescript
storage_list({
  workspace_identifier?: string,  // Either workspace...
  session_identifier?: string,    // ...or session (one required)
  path?: string                  // Optional subdirectory
})
```

### Resource Bridge Tools

These tools provide access to resource data for clients that don't support native resource reading.

#### resource_workspace_list

List all workspaces (same as `amux://workspace` resource).

```typescript
resource_workspace_list()  // No parameters
```

#### resource_workspace_show

Get workspace details (same as `amux://workspace/{id}` resource).

```typescript
resource_workspace_show({
  workspace_identifier: string  // Workspace ID, index, or name
})
```

#### resource_workspace_browse (Disabled)

**Note**: This tool has been temporarily disabled for v0.1.0 due to AI agent overuse and reliability issues. See [issue #164](https://github.com/choplin/amux/issues/164) for details.

<!--
Browse workspace files (same as `amux://workspace/{id}/files` resource).

```typescript
resource_workspace_browse({
  workspace_identifier: string,  // Workspace ID, index, or name
  path?: string                 // Optional subdirectory
})
```
-->

#### resource_session_list

List all sessions (same as `amux://session` resource).

```typescript
resource_session_list()  // No parameters
```

#### resource_session_show

Get session details (same as `amux://session/{id}` resource).

```typescript
resource_session_show({
  session_identifier: string  // Session ID, index, or name
})
```

#### resource_session_output

Get session output/logs (same as `amux://session/{id}/output` resource).

```typescript
resource_session_output({
  session_identifier: string  // Session ID, index, or name
})
```

### Prompt Tools

#### prompt_list

List all available prompts.

```typescript
prompt_list()  // No parameters
```

#### prompt_get

Get a specific prompt by name.

```typescript
prompt_get({
  name: string  // Prompt name
})
```

## Resources (Read-Only Data)

MCP Resources provide structured read-only access to Amux data.

### Workspace Resources

| URI Pattern | Description | Returns |
|-------------|-------------|---------|
| `amux://workspace` | List all workspaces | Array of workspace objects |
| `amux://workspace/{id}` | Workspace details | Single workspace with paths |
| `amux://workspace/{id}/files` | Browse workspace files | Directory listing |
| `amux://workspace/{id}/files/{path}` | Read specific file | File content |
| `amux://workspace/{id}/context` | Workspace context file | Context.md content |

### Session Resources

| URI Pattern | Description | Returns |
|-------------|-------------|---------|
| `amux://session` | List all sessions | Array of session objects |
| `amux://session/{id}` | Session details | Single session with metadata |
| `amux://session/{id}/output` | Session output/logs | Session output text |

## Prompts (Guided Workflows)

MCP Prompts guide AI agents through common workflows. Currently, prompts are primarily used internally.

### Available Prompts

- `start-issue-work` - Begin working on a GitHub issue
- `prepare-pr` - Prepare code for pull request submission
- `review-workspace` - Analyze workspace state and suggest next steps

**Note:** While prompts are registered in the MCP server, most MCP clients don't support prompts yet. Use the direct tools instead.

## Identifier Resolution

All `_identifier` parameters in MCP tools accept:

1. Full ID (e.g., `workspace-feature-auth-123`)
2. Numeric index (e.g., `1`, `2`, `3`)
3. Name (e.g., `feature-auth`)

The resolution follows this priority order: exact ID match → numeric index → name match.
