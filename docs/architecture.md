# Amux Architecture

## Overview

Amux (Agent Multiplexer) is designed to provide isolated development environments for AI agents with session
management capabilities. The architecture follows clean architecture principles with clear separation of concerns.

## Core Components

### 1. Workspace Management

**Purpose**: Manages git worktree-based isolated environments

**Components**:

- `workspace.Manager` - Core workspace operations
- Git worktree integration for isolation
- Automatic branch creation and management
- Metadata persistence in YAML files

**Key Features**:

- Each workspace is a separate git worktree
- Dedicated branches prevent conflicts
- Workspace metadata stored in `.amux/workspaces/`

### 2. Session Management

**Purpose**: Manages AI agent terminal sessions

**Components**:

- `session.Manager` - Session lifecycle management
- `session.Store` - Persistent session metadata
- `tmux.Adapter` - Terminal multiplexing backend
- Session implementations (basic and tmux-backed)

**Architecture**:

```text
SessionManager
├── SessionStore (FileStore)
│   └── YAML files in .amux/sessions/
├── TmuxAdapter
│   └── tmux process management
└── Session Cache (in-memory)
```

### 3. Configuration Management

**Purpose**: Manages project configuration including agent definitions

**Components**:

- `config.Manager` - Configuration file management
- Agent definitions in config.yaml
- Environment variable management
- Command defaults

**Configuration Structure**:

```yaml
agents:
  claude:
    name: Claude
    type: tmux
    params:
      command: claude
      environment:
        ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
```

### 4. Working Context

**Purpose**: Provides structured context files for AI agents

**Components**:

- Context file templates
- Workspace metadata tracking
- Session storage directories

**Context Files**:

- Workspace context files stored at `.amux/workspaces/{id}/context.md`
- Session storage for agent-specific data

### 5. Storage Management

**Purpose**: Provides unified file operations for workspace and session storage

**Components**:

- `storage.Manager` - Core file operations (read, write, list, remove)
- Path validation to prevent directory traversal
- Consistent error handling across CLI and MCP

**Key Features**:

- Single implementation used by both CLI commands and MCP tools
- Type-safe results for different operations
- Maintains complete functionality including operation-specific messaging

### 6. MCP Server

**Purpose**: Enables AI agents to interact with Amux

**Components**:

- MCP protocol implementation
- Tool handlers for workspace operations
- Multiple transport support (stdio, HTTP)

**Available Tools**:

- `workspace_create` - Create new workspace
- `workspace_remove` - Remove workspace
- `workspace_storage_read/write/list` - Storage operations
- `session_storage_read/write/list` - Session storage operations

**Bridge Tools** (for clients without resource support):

- `resource_workspace_list` - List all workspaces
- `resource_workspace_show` - Get workspace details
- `resource_workspace_browse` - Browse workspace files

## Data Flow

### Session Creation Flow

```text
User Command (amux run)
    ↓
CLI Command Handler
    ↓
Load Agent Config → Get Environment & Command
    ↓
Session Manager
    ↓
Create Session Info → Store in FileStore
    ↓
Initialize Context → Create template files
    ↓
Create Tmux Session → Set environment vars
    ↓
Start Agent Process → Return session ID
```

### Workspace Creation Flow

```text
User/Agent Request
    ↓
Workspace Manager
    ↓
Git Worktree Create → New branch
    ↓
Initialize Metadata → .amux/workspace.yaml
    ↓
Create Context → Template files
    ↓
Return Workspace Info
```

## Directory Structure

```text
project-root/
├── .amux/                      # Amux metadata
│   ├── config.yaml            # Project configuration
│   ├── sessions/              # Session metadata
│   │   └── session-*.yaml     # Individual session files
│   └── workspaces/            # Workspace storage
│       └── workspace-{id}/    # Workspace directories
│           ├── workspace.yaml # Workspace metadata
│           └── worktree/      # Git worktree (clean workspace)
│               └── [project files]    # Actual code
└── [main project files]       # Original repository

```

## Design Philosophy

Amux follows these architectural principles:

- **Simplicity over features** - Start simple, extend when needed
- **No external services** - Everything runs locally
- **Git-native workflows** - Leverage git's power
- **File-based storage** - Easy debugging and inspection
- **Clean interfaces** - Enable future extensibility

For detailed design decisions and rationale, see:

- [ADR-011: Git Worktrees for Workspace Isolation](adr/011-git-worktrees-for-workspace-isolation.md)
- [ADR-012: Tmux for Session Management](adr/012-tmux-for-session-management.md)
- [ADR-013: YAML for Configuration](adr/013-yaml-for-configuration.md)
- [ADR-014: File-based Session Store](adr/014-file-based-session-store.md)

## Extension Points

### 1. Backend Adapters

The session management uses an adapter pattern for backends:

```go
type Adapter interface {
    CreateSession(name, workDir string) error
    KillSession(name string) error
    SendKeys(name, keys string) error
    // ... other operations
}
```

Future backends could include:

- Docker containers
- Kubernetes pods
- Cloud-based environments

### 2. Storage Backends

The session store interface allows different implementations:

```go
type SessionStore interface {
    Save(info *SessionInfo) error
    Load(id string) (*SessionInfo, error)
    List() ([]*SessionInfo, error)
    Delete(id string) error
}
```

Future stores could include:

- SQLite for better querying
- Redis for distributed setups
- Cloud storage backends

### 3. Agent Types

The agent configuration is extensible:

```go
type Agent struct {
    Name        string
    Type        string  // claude, openai, custom
    Command     string
    Environment map[string]string
}
```

### 4. Context Templates

Context files can be customized per project or agent type by modifying the template initialization.

## Security Considerations

### 1. Environment Variables

- Sensitive values should use environment variable references
- Never commit actual API keys to config files
- Session environments are isolated

### 2. File System Access

- Workspace operations are restricted to project boundaries
- MCP file browsing validates paths
- No arbitrary command execution

### 3. Process Isolation

- Each tmux session runs as the current user
- No privilege escalation
- Standard Unix process isolation

## Performance Considerations

### 1. Workspace Creation

- Git worktree creation is I/O intensive
- Consider SSD storage for better performance
- Cleanup old workspaces regularly

### 2. Session Performance

- In-memory cache reduces file system reads
- Tmux operations are generally fast
- Monitor for orphaned sessions

### 3. File Operations

- Context file updates are append-only where possible
- YAML parsing is done on-demand
- Consider pagination for large workspace lists

## Future Enhancements

1. **Distributed Sessions** - Run agents on remote machines
2. **Session Recordings** - Full terminal session replay
3. **Resource Limits** - CPU/memory constraints per session
4. **Web UI** - Browser-based session management
5. **Metrics & Monitoring** - Session performance tracking

6. **Plugin System** - Extensible agent integrations
