# State Machine Integration Plan

## Overview

This document outlines the plan for integrating the state machine (implemented in PR #206) into the existing session management system.

## Goals

1. Replace direct status manipulation with StateManager
2. Remove StatusWorking/StatusIdle in favor of StatusRunning with separate activity tracking
3. Ensure backward compatibility for existing sessions
4. Maintain all current functionality while improving reliability

## Integration Steps

### 1. Update Session Interface

Add StateManager to the Session interface:

- Add `StateManager() *state.Manager` method
- Keep existing `Status() Status` for backward compatibility initially

### 2. Update TmuxSession Implementation

#### 2.1 Add StateManager field

- Add `stateManager *state.Manager` to tmuxSessionImpl
- Initialize in NewTmuxSession
- Create state directory under session storage

#### 2.2 Replace Status Updates

- Start(): Use StateManager transitions (Created → Starting → Running)
- Stop(): Use StateManager transitions (Running → Stopping → Stopped)
- UpdateStatus(): Use StateManager for failure detection transitions

#### 2.3 Handle Activity Tracking

- Remove StatusWorking/StatusIdle transitions
- Add separate activity tracking mechanism (e.g., LastActivityTime in session info)
- Show activity status in UI separately from lifecycle state

### 3. Update Session Manager

- Initialize StateManager when creating new sessions
- Load StateManager for existing sessions
- Ensure state persistence directory exists

### 4. Migration Strategy

For existing sessions:

1. Map old statuses to new states:
   - StatusWorking → StatusRunning
   - StatusIdle → StatusRunning
   - Others remain the same
2. Create state.json file on first load
3. Log migration for debugging

### 5. Update CLI Commands

- `amux ps`: Show lifecycle state and activity separately
- `amux status`: Include state transition history if available
- Update any status filtering to use new states

### 6. Testing Strategy

1. Unit tests for StateManager integration
2. Integration tests for session lifecycle
3. Migration tests for existing sessions
4. Backward compatibility tests

## Risks and Mitigations

### Risk 1: Breaking existing sessions

**Mitigation**: Implement careful migration logic with fallbacks

### Risk 2: Performance impact of file I/O

**Mitigation**: State is already persisted; this just adds one more file

### Risk 3: Concurrent access issues

**Mitigation**: StateManager already has mutex protection

## Success Criteria

1. All existing tests pass
2. State transitions are validated
3. Sessions recover correctly after crashes
4. Migration works seamlessly for existing sessions
5. No user-visible breaking changes

## Implementation Order

1. Add StateManager to Session interface
2. Update tmuxSessionImpl with basic integration
3. Add migration logic
4. Update session manager
5. Update CLI commands
6. Add comprehensive tests
7. Update documentation
