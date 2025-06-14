# 21. Deprecate Mailbox in Favor of General Storage

Date: 2025-06-14

## Status

Accepted

## Context

The mailbox feature was originally designed as a communication mechanism between users and AI agents running in sessions. It provided:

- Separate `in/` and `out/` directories for messages
- A mailbox-specific command structure (`mailbox send`, `mailbox recv`, etc.)
- MCP resources specifically for mailbox access

However, in practice:

1. The mailbox metaphor was overly specific and constraining
2. Agents need to store arbitrary files, not just messages
3. The separate command structure added unnecessary complexity
4. The concept implied message-passing when agents need general file storage

## Decision

We will deprecate the mailbox feature and replace it with general-purpose storage directories:

1. **Workspace Storage**: Each workspace gets a `storage/` directory for persistent data
2. **Session Storage**: Each session gets a `storage/` directory for ephemeral data
3. **MCP Storage Tools**: Provide `storage_read`, `storage_write`, and `storage_list` tools
4. **Context Migration**: Move workspace context.md into the storage directory

### New Structure

```text
.amux/
├── workspaces/
│   └── {id}/
│       ├── workspace.yaml
│       ├── worktree/         # Git worktree
│       └── storage/          # General storage (replaces context.md location)
└── sessions/
    └── {id}/
        ├── session.yaml
        └── storage/          # Session-specific storage (replaces mailbox)
```

## Consequences

### Positive

- Simpler conceptual model - just "storage" instead of "mailbox"
- More flexible - agents can store any type of file
- Cleaner API - generic read/write tools instead of mailbox-specific commands
- Better alignment with actual use cases

### Negative

- Breaking change for any existing mailbox users
- Need to migrate existing mailbox data (though adoption is minimal)
- Loss of the message-oriented structure (though this wasn't being leveraged)

### Migration

Since amux is still in alpha (v0.1.0) and the mailbox feature has minimal adoption:

1. Remove all mailbox-related code
2. Update documentation to reflect storage directories
3. No data migration needed due to minimal usage

## Implementation Details

### Code Changes

1. **Removed Components**:
   - `internal/core/mailbox/` - entire package
   - `internal/cli/commands/mailbox*.go` - all mailbox commands
   - Mailbox-related MCP resources from session resources

2. **Modified Components**:
   - `internal/core/workspace/types.go` - Changed `ContextPath` to `StoragePath`
   - `internal/core/workspace/manager.go` - Create storage directory on workspace creation
   - `internal/core/session/store.go` - Added `CreateSessionStorage()` method
   - `internal/core/session/manager.go` - Create storage directory on session start

3. **New Components**:
   - `internal/mcp/storage_tools.go` - MCP tools for storage access
   - Storage path exposure in MCP resources

### API Changes

#### Workspace Type

```go
// Before
type Workspace struct {
    ContextPath string `json:"contextPath,omitempty"`
}

// After
type Workspace struct {
    StoragePath string `json:"storagePath,omitempty"`
}

// Backward compatibility
func (w *Workspace) GetContextPath() string {
    return filepath.Join(w.StoragePath, "context.md")
}
```

#### MCP Storage Tools

```go
// storage_read - Read files from storage
// storage_write - Write files to storage
// storage_list - List storage directory contents
```

### Security Considerations

The storage tools implement path traversal protection:

- All paths are cleaned and validated
- Access is restricted to within the storage directory
- Symlinks that point outside storage are blocked
