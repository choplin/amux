# 31. Session State Machine

Date: 2025-06-24

## Status

Accepted

## Context

The current session status management in Amux has several issues:

1. **Ambiguous states**: `StatusWorking` and `StatusIdle` are based on output activity, which conflates lifecycle state with activity monitoring
2. **No transition validation**: Any status can transition to any other status without validation
3. **No crash recovery**: Session state is not persisted, making it impossible to recover after crashes
4. **Inconsistent state management**: Status updates are scattered throughout the codebase without a central authority

These issues make it difficult to implement robust features like workspace protection (preventing deletion of workspaces with active sessions) and proper session lifecycle management.

## Decision

We will implement a proper state machine for session lifecycle management with the following design:

### State Definitions

Replace the current status system with well-defined lifecycle states:

- `StatusCreated`: Session has been created but not started
- `StatusStarting`: Session is in the process of starting
- `StatusRunning`: Session is actively running
- `StatusStopping`: Session is in the process of stopping
- `StatusCompleted`: Session completed successfully
- `StatusStopped`: Session was stopped by user request
- `StatusFailed`: Session failed due to an error
- `StatusOrphaned`: Session's workspace no longer exists

### State Transitions

Only valid transitions will be allowed:

- `Created` → `Starting`, `Failed`, `Orphaned`
- `Starting` → `Running`, `Failed`, `Orphaned`
- `Running` → `Stopping`, `Completed`, `Failed`, `Orphaned`
- `Stopping` → `Stopped`, `Failed`, `Orphaned`
- Terminal states (`Completed`, `Stopped`, `Failed`, `Orphaned`) have no valid transitions

### Implementation

1. **State package**: A new `internal/core/session/state` package with:
   - `Status` type with transition validation
   - `Manager` for managing state transitions with file-based persistence
   - `ChangeHandler` support for extensibility

2. **File-based persistence**: State will be persisted to `.amux/sessions/{session-id}/state.json` for crash recovery

3. **Atomic operations**: Use atomic file rename for concurrent access safety

4. **Activity tracking**: Remove from state machine - this is session-specific logic, not generic state management

### Migration Strategy

1. First PR: Implement state machine package (no integration)
2. Second PR: Integrate state machine and migrate existing sessions
3. Third PR: Implement file-based semaphore library
4. Fourth PR: Implement workspace protection using semaphore

## Consequences

### Positive

- **Clear semantics**: Each state has a well-defined meaning in the session lifecycle
- **Reliability**: Invalid transitions are prevented at the state machine level
- **Crash recovery**: Persisted state allows recovery after process crashes
- **Extensibility**: Change handlers allow adding behavior (like semaphores) without modifying core logic
- **Testability**: State machine can be tested in isolation

### Negative

- **Breaking change**: Existing sessions will need migration (handled in second PR)
- **Increased complexity**: More code than the simple status field
- **File I/O**: Each state change requires disk write (mitigated by atomic operations)

### Future Considerations

- The state machine design supports future enhancements like:
  - Distributed locking if needed (upgrade from file-based to distributed)
  - Event sourcing for full session history
  - State-specific timeouts and policies
