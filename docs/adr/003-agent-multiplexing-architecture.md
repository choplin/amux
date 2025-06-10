# 3. Agent Multiplexing Architecture

Date: 2025-06-10

## Status

Proposed

## Context

As AI agents become more capable, users want to run multiple agent sessions concurrently within different workspaces.
Each agent should work independently while maintaining isolation. We need an architecture that enables:

- Multiple AI agent sessions running simultaneously
- Each agent working in its own isolated workspace (cave)
- Standardized context management across different AI providers
- Session management and lifecycle control

## Decision

We will implement agent multiplexing using a hybrid approach:

1. **Use tmux as the backend** for managing terminal sessions
2. **Create a Go interface layer** for session abstraction
3. **Standardize context management** using the Working Context format (background.md, plan.md, etc.)
4. **Start with Claude Code support** but design for extensibility

## Architecture

```text
┌─────────────────┐
│   CLI Layer     │
│ (agentcave cmd) │
└────────┬────────┘
         │
┌────────┴────────┐
│ Session Manager │ (Go interfaces)
│   Interface     │
└────────┬────────┘
         │
┌────────┴────────┐
│  Tmux Backend   │ (Process management)
│     Adapter     │
└────────┬────────┘
         │
┌────────┴────────────────┐
│   Agent Sessions        │
│ ┌─────┐ ┌─────┐ ┌─────┐│
│ │Cave1│ │Cave2│ │Cave3││
│ └─────┘ └─────┘ └─────┘│
└─────────────────────────┘
```

## Key Interfaces

```go
type Session interface {
    ID() string
    WorkspaceID() string
    Status() SessionStatus
    Start(ctx context.Context) error
    Stop() error
    SendInput(input string) error
    GetOutput() ([]byte, error)
}

type SessionManager interface {
    CreateSession(opts SessionOptions) (Session, error)
    GetSession(id string) (Session, error)
    ListSessions() ([]Session, error)
    RemoveSession(id string) error
}
```

## Consequences

**Benefits**:

- Multiple agents can work in parallel on different features
- Tmux provides robust session management and persistence
- Standardized context makes agent work portable
- Clean abstraction allows future backends (containers, cloud functions)

**Trade-offs**:

- Tmux dependency (though it's widely available)
- Initial implementation limited to terminal-based agents
- Need to handle session cleanup and orphaned processes

**Future Possibilities**:

- Web UI for monitoring multiple agent sessions
- Different backends for different deployment scenarios
- Cross-agent communication protocols
