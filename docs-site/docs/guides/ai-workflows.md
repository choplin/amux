---
sidebar_position: 4
---

# AI Workflows with MCP

Integrate Amux with AI assistants through the Model Context Protocol (MCP).

## Design Philosophy

Amux is designed to provide identical functionality through both CLI and MCP:

- **Core operations have MCP equivalents** - `amux ws create` → `workspace_create` tool
- **Same parameters and options** - What works in CLI works in MCP
- **Unified experience** - AI agents can perform the same meaningful operations as humans

This design ensures that workflows are portable between human and AI usage.

## Starting the MCP Server

### For Claude Code

Add to your MCP settings:

```json
{
  "mcpServers": {
    "amux": {
      "command": "/usr/local/bin/amux",
      "args": ["mcp", "--git-root", "/path/to/your/project"],
      "env": {}
    }
  }
}
```

**Important:** Always use absolute paths for both the command and `--git-root`.

### For Other MCP Clients

```bash
# Start MCP server with STDIO transport
amux mcp --git-root /path/to/your/project
```

## Available MCP Tools

### Core Operations (CLI ↔ MCP)

| Operation | CLI Command | MCP Tool |
|-----------|-------------|----------|
| Create workspace | `amux ws create <name>` | `workspace_create` |
| List workspaces | `amux ws list` | `resource_workspace_list` |
| Show workspace | `amux ws show <id>` | `resource_workspace_show` |
| Remove workspace | `amux ws remove <id>` | `workspace_remove` |
| Run agent | `amux run <agent>` | `session_run` |
| List sessions | `amux ps` | `resource_session_list` |
| Stop session | `amux session stop <id>` | `session_stop` |

### MCP-Only Features

| Feature | MCP Tool | Purpose |
|---------|----------|----------|
| Browse files | `resource_workspace_browse` | Remote file access |
| Session output | `resource_session_output` | Get logs/output |
| Send input | `session_send_input` | Interactive control |
| Storage ops | `storage_read/write/list` | Persistent data |

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

Typical workflow:
1. AI creates workspace: fix-issue-45-error-messages
2. AI reviews the issue details
3. AI implements the changes
4. AI runs tests to verify
5. AI prepares for pull request
```

### Feature Development

```text
User: "Implement user authentication with JWT"

Typical workflow:
1. AI creates workspace: feat-jwt-authentication
2. AI plans the implementation
3. AI writes the authentication module
4. AI creates tests
5. AI documents the feature
```

### Parallel AI Development

```text
User: "Run three AI agents to work on different features"

Typical workflow:
1. AI creates three workspaces for different features
2. AI starts claude in feat-api workspace
3. AI starts aider in feat-ui workspace
4. AI starts my-assistant in docs-update workspace
5. AI monitors all sessions with status checks
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
