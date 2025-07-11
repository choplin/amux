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

#### Session List

- **URI**: `amux://session`
- **Description**: List all active sessions with metadata
- **Returns**: JSON array of sessions with resource URIs

#### Session Details

- **URI**: `amux://session/{id}`
- **Description**: Get complete session information
- **Returns**: JSON object with session metadata, status, and resource URIs
- **Note**: Accepts both session ID and short ID

#### Session Output

- **URI**: `amux://session/{id}/output`
- **Description**: Read current session output/logs from tmux
- **Returns**: Plain text output from the session
- **Note**: Only available for running sessions

## MCP Tools (Actions)

Tools perform state-changing operations on workspaces.

### Core Tools

#### workspace_create

- **Description**: Create a new isolated git worktree workspace
- **Parameters**:
  - `name` (required): Workspace name
  - `description`: Workspace description
  - `branch`: Use existing branch
  - `base_branch`: Base branch to create from
  - `agent_id`: Associated agent ID
- **Returns**: Created workspace details

#### workspace_remove

- **Description**: Remove a workspace and its git worktree
- **Parameters**:
  - `workspace_id` (required): Workspace ID or name
- **Returns**: Confirmation message
- **Warning**: This operation is permanent and cannot be undone

### Storage Tools

#### storage_read

- **Description**: Read a file from workspace or session storage
- **Parameters**:
  - `workspace_id` or `session_id` (one required): ID of workspace or session
  - `path` (required): Relative path within storage directory
- **Returns**: File contents as text
- **Note**: Path validation prevents directory traversal

#### storage_write

- **Description**: Write a file to workspace or session storage
- **Parameters**:
  - `workspace_id` or `session_id` (one required): ID of workspace or session
  - `path` (required): Relative path within storage directory
  - `content` (required): Content to write to the file
- **Returns**: Success message with bytes written
- **Note**: Creates directories as needed

#### storage_list

- **Description**: List files in workspace or session storage
- **Parameters**:
  - `workspace_id` or `session_id` (one required): ID of workspace or session
  - `path` (optional): Relative path within storage directory
- **Returns**: Directory listing
- **Note**: Directories are shown with trailing slash

### Session Management Tools

#### session_run

- **Description**: Run an AI agent session in a workspace (creates and starts)
- **Parameters**:
  - `workspace_id` (required): Workspace ID or name to run in
  - `agent_id` (required): Agent ID to run (e.g., 'claude', 'gpt')
  - `command` (optional): Override the agent's default command
  - `environment` (optional): Additional environment variables
- **Returns**: Session details including ID, status, and attach commands
- **Note**: Equivalent to CLI's `amux session run`

#### session_stop

- **Description**: Stop a running agent session gracefully
- **Parameters**:
  - `session_id` (required): Session ID or short ID
- **Returns**: Confirmation message
- **Note**: Equivalent to CLI's `amux session stop`

#### session_send_input

- **Description**: Send input text to a running agent session's stdin
- **Parameters**:
  - `session_id` (required): Session ID or short ID
  - `input` (required): Input text to send to stdin
- **Returns**: Confirmation message
- **Note**: Session must be running to receive input

### Bridge Tools (Resource Access)

Many MCP clients have limited or no support for reading resources directly. To ensure compatibility, Amux provides
"bridge" tools that give tool-based access to resource data. These tools return the same data as their resource
counterparts.

#### resource_workspace_list

- **Description**: List all workspaces (bridge to `amux://workspace` resource)
- **Parameters**: None
- **Returns**: JSON array of workspaces (same as resource)
- **Note**: Use this if your MCP client cannot read resources directly

#### resource_workspace_show

- **Description**: Get workspace details (bridge to `amux://workspace/{id}` resource)
- **Parameters**:
  - `workspace_id` (required): Workspace ID or name
- **Returns**: JSON object with workspace details (same as resource)

#### resource_workspace_browse

- **Description**: Browse workspace files (bridge to `amux://workspace/{id}/files` resource)
- **Parameters**:
  - `workspace_id` (required): Workspace ID or name
  - `path` (optional): Path within workspace to browse
- **Returns**: Directory listing or file contents (same as resource)

### Bridge Tools (Session Access)

Session resources can also be accessed through bridge tools:

#### resource_session_list

- **Description**: List all active sessions (bridge to `amux://session` resource)
- **Parameters**: None
- **Returns**: JSON array of sessions (same as resource)

#### resource_session_show

- **Description**: Get session details (bridge to `amux://session/{id}` resource)
- **Parameters**:
  - `session_id` (required): Session ID or short ID
- **Returns**: JSON object with session details (same as resource)

#### resource_session_output

- **Description**: Read session output (bridge to `amux://session/{id}/output` resource)
- **Parameters**:
  - `session_id` (required): Session ID or short ID
- **Returns**: Plain text output from the session

### Bridge Tools (Prompt Access)

Similarly, prompt data can be accessed through bridge tools:

#### prompt_list

- **Description**: List all available prompts
- **Parameters**: None
- **Returns**: JSON array of prompt names and descriptions

#### prompt_get

- **Description**: Get a specific prompt definition
- **Parameters**:
  - `name` (required): Name of the prompt
- **Returns**: JSON object with prompt details including template

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

// Run an agent session
{
  "method": "tools/call",
  "params": {
    "name": "session_run",
    "arguments": {
      "workspace_id": "feature-auth",
      "agent_id": "claude",
      "command": "claude-code --model opus",
      "environment": {
        "ANTHROPIC_API_KEY": "sk-..."
      }
    }
  }
}

// Stop a session
{
  "method": "tools/call",
  "params": {
    "name": "session_stop",
    "arguments": {
      "session_id": "3"
    }
  }
}

// Send input to a session
{
  "method": "tools/call",
  "params": {
    "name": "session_send_input",
    "arguments": {
      "session_id": "3",
      "input": "Please implement the login endpoint"
    }
  }
}
```

### Using Bridge Tools

Bridge tools provide the same data as resources but through the tools interface:

```json
// List workspaces (bridge to amux://workspace)
{
  "method": "tools/call",
  "params": {
    "name": "resource_workspace_list",
    "arguments": {}
  }
}

// Get workspace details (bridge to amux://workspace/{id})
{
  "method": "tools/call",
  "params": {
    "name": "resource_workspace_show",
    "arguments": {
      "workspace_id": "ws-123"
    }
  }
}

// Browse files (bridge to amux://workspace/{id}/files)
{
  "method": "tools/call",
  "params": {
    "name": "resource_workspace_browse",
    "arguments": {
      "workspace_id": "ws-123",
      "path": "src"
    }
  }
}

// List available prompts
{
  "method": "tools/call",
  "params": {
    "name": "prompt_list",
    "arguments": {}
  }
}

// Get specific prompt
{
  "method": "tools/call",
  "params": {
    "name": "prompt_get",
    "arguments": {
      "name": "workspace_planning"
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

### AI Agent Collaboration Workflow

1. **Run an AI agent** in a workspace:

   ```json
   { "method": "tools/call", "params": { "name": "session_run", "arguments": { "workspace_id": "feature-auth", "agent_id": "claude" } } }
   ```

2. **Monitor session status** via resources:

   ```json
   { "method": "resources/read", "params": { "uri": "amux://session/3" } }
   ```

3. **Send instructions** to the agent:

   ```json
   { "method": "tools/call", "params": { "name": "session_send_input", "arguments": { "session_id": "3", "input": "Please add tests for the auth module" } } }
   ```

4. **View agent output**:

   ```json
   { "method": "resources/read", "params": { "uri": "amux://session/3/output" } }
   ```

5. **Stop the session** when done:

   ```json
   { "method": "tools/call", "params": { "name": "session_stop", "arguments": { "session_id": "3" } } }
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

Tools are implemented in:

- `internal/mcp/server.go` - Core tools (workspace_create, workspace_remove)
- `internal/mcp/bridge_tools.go` - Bridge tools for resource/prompt access

Key features:

- Type-safe parameter structs with validation
- Workspace name/ID resolution
- Proper error handling and user feedback
- Shared logic between resources and bridge tools to ensure consistency

### Prompt Implementation

Prompts are implemented in `internal/mcp/prompts.go` with:

- Dynamic content generation based on workspace state
- Structured markdown output
- Integration with workspace and git operations

## Design Decisions

1. **Resources vs Tools**: Clear separation between read operations (Resources) and state changes (Tools)
2. **Minimal Tool Set**: Only essential operations exposed as tools (create/remove)
3. **Bridge Tools**: Compatibility layer for MCP clients without resource support
4. **Path Security**: All file access validated to prevent directory traversal
5. **Name Resolution**: Both workspace IDs and names accepted for convenience
6. **Resource URIs**: Hierarchical structure for intuitive navigation
7. **Shared Logic**: Resources and bridge tools share implementation to ensure consistency

## Future Enhancements

Planned MCP extensions tracked in GitHub issues:

- Additional session management tools enhancements
- Session status change notifications via MCP events
