---
sidebar_position: 4
---

# AI Workflows with MCP

Integrate Amux with AI assistants through the Model Context Protocol (MCP).

## Design Philosophy

Amux is designed to provide identical functionality through both CLI and MCP:

- **Every CLI command has an MCP equivalent** - `amux ws create` → `workspace_create` tool
- **Same parameters and options** - What works in CLI works in MCP
- **Unified experience** - AI agents can do everything a human user can do

This design ensures that workflows are portable between human and AI usage.

## Starting the MCP Server

### For Claude Code

Add to your MCP settings:

```json
{
  "mcpServers": {
    "amux": {
      "command": "/usr/local/bin/amux",
      "args": ["mcp", "--root-dir", "/path/to/your/project"],
      "env": {}
    }
  }
}
```

**Important:** Always use absolute paths for both the command and `--root-dir`.

### For Other MCP Clients

```bash
# Start MCP server with STDIO transport
amux mcp --root-dir /path/to/your/project
```

## Available MCP Tools

### Workspace Tools

| CLI Command | MCP Tool | Purpose |
|------------|----------|----------|
| `amux ws create` | `workspace_create` | Create isolated workspace |
| `amux ws list` | `resource_workspace_list` | List all workspaces |
| `amux ws show` | `resource_workspace_show` | Show workspace details |
| `amux ws remove` | `workspace_remove` | Remove workspace |

### Session Tools

| CLI Command | MCP Tool | Purpose |
|------------|----------|----------|
| `amux run` | `session_run` | Start AI agent session |
| `amux ps` | `resource_session_list` | List running sessions |
| `amux session stop` | `session_stop` | Stop running session |
| N/A | `session_send_input` | Send input to session |

### Storage Tools

| CLI Command | MCP Tool | Purpose |
|------------|----------|----------|
| N/A | `storage_read` | Read from storage |
| N/A | `storage_write` | Write to storage |
| N/A | `storage_list` | List storage contents |

## Common Workflow Prompts

Here are effective prompts to invoke Amux tools in AI assistants:

### Starting New Work

#### "Work on issue #123"

- AI creates workspace named after the issue
- Automatically switches to the workspace
- Begins implementing the fix/feature

#### "Create a workspace for authentication feature"

- Creates `feat-authentication` workspace
- Sets up isolated environment
- Ready for development

### Managing Multiple Tasks

#### "Show me all workspaces"

- Lists workspaces with their status
- Shows which branches are active
- Helps identify work in progress

#### "What AI agents are currently running?"

- Lists all active sessions
- Shows which workspaces they're using
- Displays their current status

### Collaborative Development

#### "Start Claude in the authentication workspace"

- Runs Claude agent in specific workspace
- Keeps work isolated from other tasks
- Enables parallel development

#### "Stop the session in workspace 2"

- Identifies session in workspace
- Gracefully stops the agent
- Preserves work state

## Workflow Examples

### Issue-Based Development

```text
User: "Work on issue #45 about improving error messages"

AI Assistant:
1. Creates workspace: fix-issue-45-error-messages
2. Reviews the issue details
3. Implements the changes
4. Runs tests to verify
5. Prepares for pull request
```

### Feature Development

```text
User: "Implement user authentication with JWT"

AI Assistant:
1. Creates workspace: feat-jwt-authentication
2. Plans the implementation
3. Writes the authentication module
4. Creates tests
5. Documents the feature
```

### Parallel AI Development

```text
User: "Run three AI agents to work on different features"

AI Assistant:
1. Creates three workspaces for different features
2. Starts claude in feat-api workspace
3. Starts aider in feat-ui workspace
4. Starts my-assistant in docs-update workspace
5. Monitors all sessions with status checks
```

## Best Practices

### Workspace Naming

AI agents typically create descriptive workspace names:

- `fix-issue-123-login-bug`
- `feat-user-authentication`
- `docs-api-reference`
- `refactor-database-layer`

### Session Management

1. **One agent per workspace** - Avoid conflicts
2. **Monitor agent status** - Check for stuck sessions
3. **Clean up after completion** - Remove merged workspaces
4. **Use storage for context** - Save task-specific information

### MCP Configuration

1. **Use absolute paths** - Both for amux binary and project root
2. **Restart after changes** - MCP clients cache configuration
3. **Check initialization** - Ensure `.amux/config.yaml` exists

## Tool Reference

### workspace_create

```typescript
workspace_create({
  name: string,              // Required: workspace name
  description?: string,      // Optional: description
  branch?: string,          // Optional: use existing branch
  baseBranch?: string       // Optional: base for new branch
})
```

### session_run

```typescript
session_run({
  agent_id: string,         // Required: agent to run
  workspace_identifier?: string,  // Optional: target workspace
  name?: string,            // Optional: session name
  command?: string          // Optional: override command
})
```

### storage_write

```typescript
storage_write({
  workspace_identifier: string,  // Workspace ID or name
  path: string,                 // File path in storage
  content: string               // File content
})
```

## Troubleshooting

### Common Issues

#### "MCP server not found"

- Check amux binary path is absolute
- Verify project is initialized (`amux init`)
- Restart your MCP client

#### "Workspace not found"

- List workspaces to verify name/ID
- Ensure workspace wasn't removed
- Check for typos in identifier

#### "Session failed to start"

- Verify agent is configured
- Check workspace exists
- Review agent command in config
