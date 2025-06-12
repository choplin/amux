# MCP Usage Examples

This document demonstrates how AI agents can use Amux's MCP features.

## Resources (Read-only Data)

### List All Workspaces

```json
// Request
{
  "method": "resources/read",
  "params": {
    "uri": "amux://workspace"
  }
}

// Response
{
  "contents": [{
    "uri": "amux://workspace",
    "mimeType": "application/json",
    "text": "[{\"id\":\"ws-123\",\"name\":\"feature-auth\",\"branch\":\"feat/auth\",\"resources\":{\"detail\":\"amux://workspace/ws-123\",\"files\":\"amux://workspace/ws-123/files\",\"context\":\"amux://workspace/ws-123/context\"}}]"
  }]
}
```

### Get Workspace Details

```json
// Request
{
  "method": "resources/read",
  "params": {
    "uri": "amux://workspace/ws-123"
  }
}

// Response includes paths for direct filesystem access
{
  "contents": [{
    "uri": "amux://workspace/ws-123",
    "mimeType": "application/json",
    "text": "{\"id\":\"ws-123\",\"name\":\"feature-auth\",\"paths\":{\"worktree\":\"/path/to/worktree\",\"context\":\"/path/to/context.md\"},\"resources\":{...}}"
  }]
}
```

### Browse Files

```json
// List directory
{
  "method": "resources/read",
  "params": {
    "uri": "amux://workspace/ws-123/files/src"
  }
}

// Read specific file
{
  "method": "resources/read",
  "params": {
    "uri": "amux://workspace/ws-123/files/src/main.go"
  }
}
```

## Prompts (Guided Workflows)

### Start Working on an Issue

```json
// Request
{
  "method": "prompts/get",
  "params": {
    "name": "start-issue-work",
    "arguments": {
      "issue_number": "42",
      "issue_title": "Add authentication system",
      "issue_url": "https://github.com/user/repo/issues/42"
    }
  }
}

// Response provides structured workflow
{
  "messages": [
    {
      "role": "user",
      "content": "# Starting Work on Issue #42: Add authentication system\n\n## Issue Details\n- Number: #42\n- Title: Add authentication system\n- URL: https://github.com/user/repo/issues/42\n\n## Workflow Steps\n\n1. **Understand Requirements**..."
    }
  ]
}
```

### Prepare Pull Request

```json
// Request
{
  "method": "prompts/get",
  "params": {
    "name": "prepare-pr",
    "arguments": {
      "workspace_id": "ws-123",
      "pr_title": "feat: implement JWT authentication",
      "pr_description": "This PR adds JWT-based authentication..."
    }
  }
}
```

### Review Workspace State

```json
// Request
{
  "method": "prompts/get",
  "params": {
    "name": "review-workspace",
    "arguments": {
      "workspace_id": "ws-123"
    }
  }
}
```

## Tools (Actions)

### Create Workspace

```json
// Request
{
  "method": "tools/call",
  "params": {
    "name": "workspace_create",
    "arguments": {
      "name": "feature-auth",
      "description": "Implement authentication system"
    }
  }
}
```

### Remove Workspace

```json
// Request
{
  "method": "tools/call",
  "params": {
    "name": "workspace_remove",
    "arguments": {
      "workspace_id": "ws-123"
    }
  }
}
```

## Typical Workflow

1. **Start with a prompt** to get guided workflow:

   ```json
   { "method": "prompts/get", "params": { "name": "start-issue-work", "arguments": { "issue_number": "42" } } }
   ```

2. **Create a workspace** using a tool:

   ```json
   { "method": "tools/call", "params": { "name": "workspace_create", "arguments": { "name": "feature-auth" } } }
   ```

3. **Read workspace details** to get paths:

   ```json
   { "method": "resources/read", "params": { "uri": "amux://workspace/ws-123" } }
   ```

4. **Browse and read files** as needed:

   ```json
   { "method": "resources/read", "params": { "uri": "amux://workspace/ws-123/files" } }
   ```

5. **Review progress** periodically:

   ```json
   { "method": "prompts/get", "params": { "name": "review-workspace", "arguments": { "workspace_id": "ws-123" } } }
   ```

6. **Prepare PR** when done:

   ```json
   { "method": "prompts/get", "params": { "name": "prepare-pr", "arguments": { "workspace_id": "ws-123" } } }

   ```
