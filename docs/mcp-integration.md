# MCP Integration Guide

Amux provides full Model Context Protocol (MCP) support, enabling AI agents to manage workspaces and sessions programmatically.

## Overview

The MCP server exposes Amux functionality through:

- **Resources** - Read-only access to workspace data
- **Tools** - Actions to create and manage workspaces
- **Prompts** - Guided workflows for common tasks

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
# STDIO transport (default)
amux mcp

# HTTPS transport
amux mcp --transport https --port 3000 --auth bearer --token your-secret
```

## MCP Resources

Resources provide read-only access to Amux data.

### Workspace List

```text
GET amux://workspace
```

Returns all workspaces with metadata:

```json
{
  "workspaces": [
    {
      "id": "workspace-feature-auth-123",
      "index": "1",
      "name": "feature-auth",
      "branch": "feature/auth",
      "created": "2024-01-15T10:00:00Z",
      "resources": [
        "amux://workspace/workspace-feature-auth-123",
        "amux://workspace/workspace-feature-auth-123/files",
        "amux://workspace/workspace-feature-auth-123/context"
      ]
    }
  ]
}
```

### Workspace Details

```text
GET amux://workspace/{id}
```

Returns detailed workspace information:

```json
{
  "id": "workspace-feature-auth-123",
  "name": "feature-auth",
  "branch": "feature/auth",
  "baseBranch": "main",
  "path": "/project/.amux/workspaces/workspace-feature-auth-123/worktree",
  "storagePath": "/project/.amux/workspaces/workspace-feature-auth-123/storage",
  "contextPath": "/project/.amux/workspaces/workspace-feature-auth-123/context.md",
  "description": "Implement authentication system",
  "created": "2024-01-15T10:00:00Z"
}
```

### File Browser

```text
GET amux://workspace/{id}/files
GET amux://workspace/{id}/files/{path}
```

Browse workspace files:

```json
{
  "path": "src/",
  "entries": [
    {"name": "auth.go", "type": "file", "size": 1234},
    {"name": "handlers/", "type": "directory"}
  ]
}
```

### Workspace Context

```text
GET amux://workspace/{id}/context
```

Access the workspace's context.md file for task documentation.

### Session Resources

```text
GET amux://session
GET amux://session/{id}
GET amux://session/{id}/output
```

Access session information and logs.

## MCP Tools

Tools allow AI agents to perform actions.

### workspace_create

Create a new isolated workspace.

**Parameters:**

- `name` (required) - Workspace name
- `description` (optional) - Workspace description
- `branch` (optional) - Use existing branch
- `baseBranch` (optional) - Base branch for new workspace

**Example:**

```javascript
workspace_create({
  name: "feature-auth",
  description: "Implement JWT authentication",
  baseBranch: "main"
})
```

### workspace_remove

Remove a workspace and its worktree.

**Parameters:**

- `workspace_identifier` (required) - Workspace ID or name

**Example:**

```javascript
workspace_remove({
  workspace_identifier: "feature-auth"
})
```

### session_run

Start an AI agent session.

**Parameters:**

- `agent_id` (required) - Agent identifier
- `workspace_identifier` (optional) - Target workspace
- `name` (optional) - Session name
- `command` (optional) - Override agent command

**Example:**

```javascript
session_run({
  agent_id: "claude",
  workspace_identifier: "feature-auth",
  name: "auth-implementation"
})
```

### session_stop

Stop a running session.

**Parameters:**

- `session_identifier` (required) - Session ID

### session_send_input

Send input to a running session.

**Parameters:**

- `session_identifier` (required) - Session ID
- `input` (required) - Text to send

### storage_read / storage_write

Access workspace or session storage.

**Parameters:**

- `workspace_identifier` or `session_identifier` (required)
- `path` (required) - File path within storage
- `content` (required for write) - File content

## MCP Prompts

Prompts guide AI agents through common workflows.

### start-issue-work

Begin working on a GitHub issue.

**Parameters:**

- `issue_number` (required) - Issue number
- `issue_title` (optional) - Issue title
- `issue_url` (optional) - GitHub issue URL

**Flow:**

1. Creates appropriate workspace
2. Sets up context with issue details
3. Guides through implementation planning

### prepare-pr

Prepare code for pull request submission.

**Parameters:**

- `pr_title` (optional) - PR title
- `pr_description` (optional) - PR description

**Flow:**

1. Runs tests
2. Checks code formatting
3. Reviews changes
4. Helps create PR

### review-workspace

Analyze workspace state and suggest next steps.

**Parameters:**

- `workspace_id` (required) - Workspace to review

**Flow:**

1. Checks workspace age and status
2. Reviews uncommitted changes
3. Suggests appropriate actions

## Best Practices

### For AI Agents

1. **Always specify baseBranch** when creating workspaces
2. **Use descriptive workspace names** with prefixes (feat-, fix-, etc.)
3. **Clean up workspaces** after PR merge
4. **Check workspace existence** before creating

### For Developers

1. **Use absolute paths** in MCP configuration
2. **Restart Claude Code** after config changes
3. **Monitor sessions** with `amux ps` command
4. **Use workspace storage** for agent-specific files

## Example Workflows

### Feature Development

```javascript
// 1. Create workspace for new feature
const ws = await workspace_create({
  name: "feat-user-profiles",
  description: "Add user profile management",
  baseBranch: "develop"
});

// 2. Start agent session
await session_run({
  agent_id: "claude",
  workspace_identifier: ws.id
});

// 3. Work on feature...

// 4. Prepare for PR
await prompt("prepare-pr", {
  pr_title: "feat: add user profile management"
});
```

### Bug Fix

```javascript
// 1. Use existing branch
await workspace_create({
  name: "fix-login-bug",
  branch: "hotfix/login-issue",
  description: "Fix login validation bug"
});

// 2. Quick fix workflow
await prompt("start-issue-work", {
  issue_number: "123",
  issue_title: "Login fails with special characters"
});
```

### Parallel Development

```javascript
// Create multiple workspaces
const workspaces = await Promise.all([
  workspace_create({ name: "feat-api-v2" }),
  workspace_create({ name: "fix-memory-leak" }),
  workspace_create({ name: "docs-update" })
]);

// Run different agents in each
await Promise.all([
  session_run({ agent_id: "claude", workspace_identifier: workspaces[0].id }),
  session_run({ agent_id: "gpt", workspace_identifier: workspaces[1].id }),
  session_run({ agent_id: "gemini", workspace_identifier: workspaces[2].id })
]);
```

## Troubleshooting

### Connection Issues

1. Verify `amux` binary path is absolute
2. Ensure `--git-root` points to initialized project
3. Check `.amux/config.yaml` exists
4. Restart MCP client after config changes

### Resource Access

- Resources use `amux://` URI scheme
- IDs can be full ID or numeric index
- File paths are relative to workspace root

### Tool Errors

Common error codes:

- `-32602` - Invalid parameters
- `-32603` - Internal error (check logs)
- `-32001` - Resource not found
- `-32002` - Already exists
