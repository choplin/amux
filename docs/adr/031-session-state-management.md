# 31. Session State Management

Date: 2025-06-24

## Status

Accepted

## Context

Session lifecycle management in Amux requires reliable state tracking across multiple processes and potential system restarts. We needed a solution that would:

1. Track session states accurately across their lifecycle
2. Ensure atomic state transitions with proper validation
3. Support inter-process coordination for workspace semaphore functionality
4. Detect and handle session failures, completion, and orphaned states
5. Track activity metrics to help implementations determine if sessions need attention
6. Persist state information for recovery after crashes

The initial implementation used simple in-memory state tracking, which was insufficient for production use. We also needed to balance between providing useful information (like idle detection) while keeping the state model simple and focused on actual lifecycle states rather than transient conditions.

## Decision

We implemented a file-based state management system with the following architecture:

### State Manager Pattern

We created a dedicated `state` package with a `Manager` type (not "StateMachine" to avoid exposing implementation details) that handles all state transitions and persistence. This provides:

- **Atomic Operations**: State transitions use file locking to ensure atomicity across processes
- **State Validation**: Valid state transitions are enforced through a transition map
- **Activity Metrics**: Records facts about session activity (last output hash, time) without interpreting them
- **Change Notifications**: Observer pattern for state change handlers
- **Persistent Storage**: State is persisted to `{session_dir}/state.json` files

### Simplified State Model

We use a simplified state model focusing on lifecycle states:

- `created`: Session has been created but not started
- `starting`: Session is in the process of starting (acquiring resources, creating tmux session)
- `running`: Session is active and executing
- `stopping`: Session is in the process of stopping
- `stopped`: Session has been stopped by user
- `failed`: Session failed to start or crashed
- `completed`: Session finished successfully (command exited with code 0)
- `orphaned`: Session's workspace was deleted while session was active

### State Transition Rules

Valid transitions are explicitly defined:

- `created` → `starting`, `failed`, `orphaned`
- `starting` → `running`, `failed`, `orphaned`
- `running` → `stopping`, `failed`, `completed`, `orphaned`
- `stopping` → `stopped`, `failed`
- Terminal states (`stopped`, `failed`, `completed`, `orphaned`) have no outgoing transitions

### Activity Tracking vs State

The state manager tracks activity metrics but does not interpret them:

```go
type Data struct {
    State          Status    `json:"state"`
    StateChangedAt time.Time `json:"state_changed_at"`
    UpdatedBy      int       `json:"updated_by"`
    SessionID      string    `json:"session_id"`
    WorkspaceID    string    `json:"workspace_id"`

    // Activity measurements (facts, not interpretations)
    LastActivityHash uint32    `json:"last_activity_hash,omitempty"`
    LastActivityAt   time.Time `json:"last_activity_at,omitempty"`
    LastCheckedAt    time.Time `json:"last_checked_at,omitempty"`
}
```

Session implementations decide what the activity data means based on their context (e.g., a REPL might consider 30 seconds as idle, while a compilation might need 5 minutes).

### UI Presentation

The UI presents activity information as attributes of running sessions rather than separate states:

```text
SESSION  NAME    DESCRIPTION        AGENT    WORKSPACE    STATUS                    IN STATUS  TOTAL TIME
1        dev     Development env    claude   fix-auth     running (idle 2m 15s)     3m 5s      15m 23s
2        build   Building project   bash     main         running                   45s        45s
```

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

### Implementation Approach

The implementation was done in phases:

1. **Phase 1**: Created state machine with full state model including working/idle states
2. **Phase 2**: Refactored to separate concerns - state manager tracks facts, not interpretations
3. **Phase 3**: Created dedicated package structure for better visibility control
4. **Phase 4**: Simplified state model by removing working/idle states entirely

This iterative approach allowed us to validate the core mechanics before simplifying to the final design.

## Consequences

### Positive

- **Reliability**: State persists across process restarts and crashes
- **Consistency**: Atomic operations prevent race conditions
- **Simplicity**: Fewer states make the system easier to understand and test
- **Flexibility**: Session types can interpret activity data according to their needs
- **Separation of Concerns**: State management records facts; implementations make interpretations
- **Inter-process Coordination**: Multiple processes can safely coordinate

### Negative

- **File I/O**: Each state change requires disk writes
- **Debugging**: State stored in files requires additional tooling to inspect
- **Migration**: Future state schema changes require migration logic
- **Implementation Logic**: Each session type needs its own activity interpretation logic

### Trade-offs

We chose file-based persistence over alternatives like:

- **Database**: Too heavy for CLI tool, adds deployment complexity
- **Shared Memory**: Platform-specific, doesn't survive restarts
- **In-Memory Only**: Can't coordinate between processes or survive crashes

The file-based approach provides the right balance of simplicity, reliability, and portability for a CLI tool while enabling critical features like workspace semaphores.
