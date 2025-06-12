# Model Context Protocol (MCP) Integration

Amux implements the Model Context Protocol to provide AI agents with structured access to workspace data and operations.

## Architecture Overview

The MCP implementation in Amux follows a clean separation of concerns:

- **Resources** - Read-only data access
- **Tools** - State-changing operations
- **Prompts** - Guided workflows

## MCP Resources (Read-only Data)

Resources provide structured read-only access to workspace information without modifying state.

### Static Resources

#### Workspace List
- **URI**: `amux://workspace`
- **Description**: List all amux workspaces with metadata
- **Returns**: JSON array of workspaces with resource URIs

### Dynamic Resources

#### Workspace Details
- **URI**: `amux://workspace/{id}`
- **Description**: Get complete workspace information
- **Returns**: JSON object with workspace metadata, paths, and resource URIs
- **Note**: Accepts both workspace ID and name

#### Workspace Files
- **URI**: `amux://workspace/{id}/files`
- **URI**: `amux://workspace/{id}/files/{path}`
- **Description**: Browse directories or read specific files
- **Returns**:
  - Directory: JSON array of file entries
  - File: File contents with MIME type detection
- **Security**: Path validation prevents traversal attacks

#### Workspace Context
- **URI**: `amux://workspace/{id}/context`
- **Description**: Read the workspace's context.md file
- **Returns**: Markdown content or placeholder if not found

## MCP Tools (Actions)

Tools perform state-changing operations on workspaces.

### workspace_create
- **Description**: Create a new isolated git worktree workspace
- **Parameters**:
  - `name` (required): Workspace name
  - `description`: Workspace description
  - `branch`: Use existing branch
  - `base_branch`: Base branch to create from
  - `agent_id`: Associated agent ID
- **Returns**: Created workspace details

### workspace_remove
- **Description**: Remove a workspace and its git worktree
- **Parameters**:
  - `workspace_id` (required): Workspace ID or name
- **Returns**: Confirmation message
- **Warning**: This operation is permanent and cannot be undone

## MCP Prompts (Guided Workflows)

Prompts provide structured guidance for common AI agent workflows.

### start-issue-work
- **Description**: Guide through starting work on an issue
- **Arguments**:
  - `issue_number` (required): Issue number to work on
  - `issue_title`: Title of the issue
  - `issue_url`: URL of the issue
- **Provides**:
  - Structured workflow steps
  - Requirements clarification guidance
  - Planning templates

### prepare-pr
- **Description**: Prepare code for pull request submission
- **Arguments**:
  - `workspace_id` (required): Workspace ID or name
  - `pr_title`: Suggested PR title
  - `pr_description`: Suggested PR description
- **Provides**:
  - Test verification checklist
  - Code formatting reminders
  - PR creation commands

### review-workspace
- **Description**: Analyze workspace state and suggest next steps
- **Arguments**:
  - `workspace_id` (required): Workspace ID or name
- **Provides**:
  - Workspace age analysis
  - Branch status review
  - Recommended actions based on state

## Usage Examples

### Reading Resources

```json
// List all workspaces
{
  "method": "resources/read",
  "params": {
    "uri": "amux://workspace"
  }
}

// Get specific workspace details
{
  "method": "resources/read",
  "params": {
    "uri": "amux://workspace/ws-feature-auth-123"
  }
}

// Browse workspace files
{
  "method": "resources/read",
  "params": {
    "uri": "amux://workspace/ws-123/files/src"
  }
}

// Read a specific file
{
  "method": "resources/read",
  "params": {
    "uri": "amux://workspace/ws-123/files/src/main.go"
  }
}

// Read workspace context
{
  "method": "resources/read",
  "params": {
    "uri": "amux://workspace/ws-123/context"
  }
}
```

### Using Tools

```json
// Create a new workspace
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

// Remove a workspace
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

### Using Prompts

```json
// Start working on an issue
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

// Prepare a pull request
{
  "method": "prompts/get",
  "params": {
    "name": "prepare-pr",
    "arguments": {
      "workspace_id": "ws-123",
      "pr_title": "feat: implement JWT authentication"
    }
  }
}

// Review workspace state
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

## Typical AI Agent Workflow

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

## Implementation Details

### Resource Implementation

Resources are implemented in:
- `internal/mcp/resources.go` - Static resources
- `internal/mcp/resource_templates.go` - Dynamic resources with URI templates

Key features:
- URI template matching using RFC 6570 patterns
- Security validation to prevent path traversal
- MIME type detection for file contents
- Workspace resolution by both ID and name

### Tool Implementation

Tools are implemented in `internal/mcp/server.go` with:
- Type-safe parameter structs with validation
- Workspace name/ID resolution
- Proper error handling and user feedback

### Prompt Implementation

Prompts are implemented in `internal/mcp/prompts.go` with:
- Dynamic content generation based on workspace state
- Structured markdown output
- Integration with workspace and git operations

## Design Decisions

1. **Resources vs Tools**: Clear separation between read operations (Resources) and state changes (Tools)
2. **Minimal Tool Set**: Only essential operations exposed as tools (create/remove)
3. **Path Security**: All file access validated to prevent directory traversal
4. **Name Resolution**: Both workspace IDs and names accepted for convenience
5. **Resource URIs**: Hierarchical structure for intuitive navigation

## Future Enhancements

Planned MCP extensions tracked in GitHub issues:
- [#53](https://github.com/choplin/amux/issues/53): MCP Resources for session management

- [#54](https://github.com/choplin/amux/issues/54): MCP Resources and Tools for mailbox system