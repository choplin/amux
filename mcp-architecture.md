# Amux MCP Architecture Design

## Overview

This document outlines a comprehensive MCP architecture for amux that properly utilizes Resources, Prompts, Tools,
and potentially Sampling to create a complete agent multiplexing experience.

## Current State

Currently, amux only implements Tools:

- `workspace_create`
- `workspace_list`
- `workspace_get`
- `workspace_remove`
- `workspace_info`

## Proposed MCP Architecture

### 1. Resources (Read-only Data Access)

Resources expose amux data that AI agents can read and use as context.

```typescript
// Workspace Resources
"amux://workspace/{id}/files/{path}"      // Browse files in workspace
"amux://workspace/{id}/context"           // Read workspace context (context.md)
"amux://workspace/{id}/git/status"        // Git status of workspace
"amux://workspace/{id}/git/diff"          // Git diff of workspace

// Session Resources (when implemented)
"amux://session/{id}/output"              // Read session output/logs
"amux://session/{id}/status"              // Session status (running/stopped/etc)
"amux://session/{id}/mailbox/inbox"       // Read messages sent to session
"amux://session/{id}/mailbox/outbox"      // Read messages from session

// System Resources
"amux://conventions"                      // Amux conventions and guidelines
"amux://workspace"                        // List all workspaces (template)
"amux://session"                          // List all sessions (template)
```

### 2. Tools (State-changing Operations)

Tools perform actions that modify state.

```typescript
// Workspace Management
workspace_create(name, baseBranch, description?)
workspace_remove(workspace_id)

// Session Management (future)
session_start(agent, workspace_id)
session_stop(session_id)
session_attach(session_id)

// Mailbox Operations
mailbox_send(session_id, message)
mailbox_clear(session_id, direction)

// Git Operations (future)
workspace_commit(workspace_id, message)
workspace_push(workspace_id)
```

### 3. Prompts (User-initiated Workflows)

Prompts provide guided workflows for common tasks.

```yaml
work-on-issue:
  description: "Start working on a GitHub issue"
  arguments:
    - issue_number: "Issue number to work on"
    - base_branch: "Base branch (default: main)"
  flow:
    1. Fetch issue details
    2. Create workspace with appropriate name
    3. Set up context.md with issue information
    4. Return workspace info

prepare-pr:
  description: "Prepare workspace for pull request"
  arguments:
    - workspace_id: "Workspace to prepare"
  flow:
    1. Check git status
    2. Run tests (just test)
    3. Run linting (just check)
    4. Show diff summary
    5. Suggest commit message

review-session:
  description: "Review session output and provide guidance"
  arguments:
    - session_id: "Session to review"
  flow:
    1. Get session output
    2. Check mailbox messages
    3. Analyze progress
    4. Suggest next steps

cleanup-workspace:
  description: "Clean up after PR is merged"
  arguments:
    - workspace_id: "Workspace to clean"
    - pr_number: "Merged PR number"
  flow:
    1. Verify PR is merged
    2. Archive context
    3. Remove workspace
    4. Update issue status
```

### 4. Conventions Resource

The `amux://conventions` resource provides essential information:

```json
{
  "paths": {
    "workspace_root": ".amux/workspaces/{workspace-id}/worktree/",
    "workspace_context": ".amux/workspaces/{workspace-id}/context.md",
    "session_mailbox": ".amux/mailbox/{session-id}/",
    "mailbox_inbox": ".amux/mailbox/{session-id}/in/",
    "mailbox_outbox": ".amux/mailbox/{session-id}/out/"
  },
  "patterns": {
    "branch_name": "amux/workspace-{name}-{timestamp}-{hash}",
    "workspace_id": "workspace-{name}-{timestamp}-{hash}",
    "session_id": "session-{index}"
  },
  "workflows": {
    "issue_branch_prefix": {
      "bug": "fix-issue-",
      "feature": "feat-",
      "chore": "chore-"
    },
    "commit_style": "conventional",
    "pr_style": "draft_first"
  },
  "commands": {
    "test": "just test",
    "lint": "just check",
    "build": "just build",
    "format": "just fmt"
  }
}
```

## Integration Example

Here's how an AI agent would use the complete MCP architecture:

```typescript
// 1. User: "Work on issue #45"
// AI uses the work-on-issue prompt
const result = await mcp.prompt.execute("work-on-issue", {
  issue_number: 45,
  base_branch: "main"
});

// 2. AI reads conventions to understand structure
const conventions = await mcp.resource.read("amux://conventions");

// 3. AI browses workspace files
const files = await mcp.resource.read("amux://workspace/1/files/");

// 4. AI reads workspace context
const context = await mcp.resource.read("amux://workspace/1/context");

// 5. AI makes changes and prepares PR
const prReady = await mcp.prompt.execute("prepare-pr", {
  workspace_id: "1"
});

// 6. After PR is merged, cleanup
await mcp.prompt.execute("cleanup-workspace", {
  workspace_id: "1",
  pr_number: 45
});
```

## Benefits of This Architecture

1. **Clear Separation**: Resources for reading, Tools for writing, Prompts for workflows
2. **Discoverable**: AI agents can list available resources and understand conventions
3. **Guided Workflows**: Prompts provide structure for complex tasks
4. **Extensible**: Easy to add new resources, tools, and prompts
5. **Self-documenting**: Conventions resource explains how to use amux

## Implementation Priority

1. **Phase 1**: Resources for workspaces and conventions
2. **Phase 2**: Prompts for common workflows
3. **Phase 3**: Session resources when session management is complete
4. **Phase 4**: Advanced git operation tools

## Questions to Consider

1. Should we version the conventions resource?
2. Do we need sampling for any server-initiated AI tasks?
3. Should resources support watching/subscriptions for changes?
4. How do we handle resource permissions/access control?
