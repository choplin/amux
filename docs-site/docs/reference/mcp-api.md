---
sidebar_position: 3
---

# MCP API Reference

Complete reference for Amux MCP protocol implementation.

## Resources

### amux://workspace

List all workspaces.

**Response:**

```json
{
  "workspaces": [
    {
      "id": "workspace-feature-auth-123",
      "index": "1",
      "name": "feature-auth",
      "branch": "amux/workspace-feature-auth-123",
      "created": "2024-01-15T10:00:00Z",
      "resources": [...]
    }
  ]
}
```

### amux://workspace/\{id\}

Get workspace details.

**Parameters:**

- `id` - Workspace ID or name

**Response:**

```json
{
  "id": "workspace-feature-auth-123",
  "name": "feature-auth",
  "branch": "amux/workspace-feature-auth-123",
  "baseBranch": "main",
  "path": "/project/.amux/workspaces/...",
  "storagePath": "/project/.amux/workspaces/.../storage",
  "contextPath": "/project/.amux/workspaces/.../context.md",
  "description": "Implement authentication",
  "created": "2024-01-15T10:00:00Z"
}
```

### amux://workspace/\{id\}/files

Browse workspace files.

**Parameters:**

- `id` - Workspace ID
- `path` (optional) - Subdirectory path

### amux://session

List all sessions.

### amux://session/\{id\}

Get session details.

### amux://session/\{id\}/output

Get session output logs.

## Tools

### workspace_create

Create new workspace.

**Parameters:**

```typescript
{
  name: string;           // Required
  description?: string;
  branch?: string;        // Use existing branch
  baseBranch?: string;    // Base for new branch
}
```

### workspace_remove

Remove workspace.

**Parameters:**

```typescript
{
  workspace_identifier: string;  // ID or name
}
```

### session_run

Start agent session.

**Parameters:**

```typescript
{
  agent_id: string;
  workspace_identifier?: string;
  name?: string;
  command?: string;
}
```

### session_stop

Stop running session.

**Parameters:**

```typescript
{
  session_identifier: string;
}
```

### session_send_input

Send input to session.

**Parameters:**

```typescript
{
  session_identifier: string;
  input: string;
}
```

### storage_read

Read from storage.

**Parameters:**

```typescript
{
  workspace_identifier?: string;
  session_identifier?: string;
  path: string;
}
```

### storage_write

Write to storage.

**Parameters:**

```typescript
{
  workspace_identifier?: string;
  session_identifier?: string;
  path: string;
  content: string;
}
```

## Prompts

### start-issue-work

**Parameters:**

```typescript
{
  issue_number: string;     // Required
  issue_title?: string;
  issue_url?: string;
}
```

### prepare-pr

**Parameters:**

```typescript
{
  pr_title?: string;
  pr_description?: string;
}
```

### review-workspace

**Parameters:**

```typescript
{
  workspace_id: string;     // Required
}
```

## Error Codes

- `-32602` - Invalid parameters
- `-32603` - Internal error
- `-32001` - Resource not found
- `-32002` - Already exists
- `-32003` - Permission denied

## Transport Options

### STDIO (Default)

```bash
amux mcp
```

### HTTPS

```bash
amux mcp --transport https --port 3000
```

### Authentication

```bash
amux mcp --auth bearer --token SECRET
```
