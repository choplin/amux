# 31. Session State Management

Date: 2025-06-24

## Status

Accepted

## Context

Session lifecycle management in Amux requires reliable state tracking across multiple processes and potential system restarts. We needed a solution that would:

1. Track session states accurately (created, starting, running, working, idle, stopping, stopped, failed, orphaned)
2. Ensure atomic state transitions with proper validation
3. Support inter-process coordination for workspace semaphore functionality
4. Detect and handle session failures, completion, and orphaned states
5. Track activity to differentiate between working/idle states
6. Persist state information for recovery after crashes

The initial implementation used simple in-memory state tracking, which was insufficient for production use.

## Decision

We implemented a file-based state management system with the following architecture:

### State Manager Pattern

We created a dedicated `state` package with a `Manager` type (not "StateMachine" to avoid exposing implementation details) that handles all state transitions and persistence. This provides:

- **Atomic Operations**: State transitions use file locking to ensure atomicity across processes
- **State Validation**: Valid state transitions are enforced through a transition map
- **Activity Tracking**: Tracks last output hash, output time, and status check time
- **Change Notifications**: Observer pattern for state change handlers
- **Persistent Storage**: State is persisted to `{session_dir}/state.json` files

### State Transition Rules

Valid transitions are explicitly defined:

- `created` → `starting`
- `starting` → `running`, `failed`
- `running` → `working`, `idle`, `stopping`, `failed`
- `working` → `idle`, `stopping`, `failed`, `completed`
- `idle` → `working`, `stopping`, `failed`
- `stopping` → `stopped`, `failed`

### Package Structure

```text
internal/core/session/
├── state/
│   ├── manager.go      # Core state management logic
│   ├── types.go        # Status constants and Data structures
│   └── logger.go       # Logger interface for package
└── types.go            # Re-exports state types for API compatibility
```

### Key Implementation Details

1. **File-based Locking**: Uses `flock` for inter-process coordination
2. **Atomic File Updates**: Write to temp file then atomic rename
3. **Activity Detection**: Compares output hashes to detect activity
4. **Process Monitoring**: Checks tmux session existence and shell process status
5. **Graceful Degradation**: Continues with current state if updates fail

## Consequences

### Positive

- **Reliability**: State persists across process restarts and crashes
- **Consistency**: Atomic operations prevent race conditions
- **Observability**: Clear state transitions with logging
- **Extensibility**: Easy to add new states or transition rules
- **Separation of Concerns**: State management isolated from session logic
- **Inter-process Coordination**: Multiple processes can safely coordinate

### Negative

- **Complexity**: More complex than in-memory state tracking
- **File I/O**: Each state change requires disk writes
- **Debugging**: State stored in files requires additional tooling to inspect
- **Migration**: Future state schema changes require migration logic

### Trade-offs

We chose file-based persistence over alternatives like:

- **Database**: Too heavy for CLI tool, adds deployment complexity
- **Shared Memory**: Platform-specific, doesn't survive restarts
- **In-Memory Only**: Can't coordinate between processes or survive crashes

The file-based approach provides the right balance of simplicity, reliability, and portability for a CLI tool while enabling critical features like workspace semaphores.
