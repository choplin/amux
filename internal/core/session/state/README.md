# Session State Management

This package implements a proper state machine for managing session lifecycle transitions.

## What This PR Includes

✅ **Core State Machine Implementation**

- Status type with predefined states (Created, Starting, Running, Stopping, Completed, Stopped, Failed, Orphaned)
- State transition validation rules
- StateManager for managing transitions with persistence
- State change handler support for side effects
- Comprehensive test coverage (100% coverage)

## Integration Plan (Future PRs)

### 1. Complete TmuxSession Integration

The tmux_session.go file still has many references to the old StatusState field that need to be updated:

- Replace all `s.info.StatusState.Status` with StateManager calls
- Update Start() to use StateManager.TransitionTo() instead of direct status updates
- Update Stop() to use StateManager.TransitionTo()
- Update SendInput() to use StateManager.UpdateActivity()
- Update UpdateStatus() to use StateManager for all status changes

### 2. Update Session Manager

- Initialize StateManager when creating sessions
- Pass StateManager to TmuxSession via WithStateManager option
- Update any other session implementations

### 3. Update UI Components

- Update CLI commands that display status
- Remove references to StatusWorking and StatusIdle
- Show "idle" as an attribute based on LastActivityTime instead of as a status

### 4. Migration Considerations

- Handle existing sessions that have StatusState in their persisted YAML
- Provide migration logic to convert old status to new states

## State Transition Rules

```text
Created → Starting, Failed, Orphaned
Starting → Running, Failed, Orphaned
Running → Stopping, Completed, Failed, Orphaned
Stopping → Stopped, Failed, Orphaned
Completed, Stopped, Failed, Orphaned → (terminal states, no transitions)
```

## Usage Example

```go
// Create state manager
stateManager := state.NewManager(
    sessionID,
    workspaceID,
    stateDir,
    logger,
)

// Add handlers for side effects
stateManager.AddStateChangeHandler(semaphoreHandler)

// Transition states
err := stateManager.TransitionTo(ctx, state.StatusStarting)

// Check current state
currentState, err := stateManager.CurrentState()

// Update activity without changing state
err = stateManager.UpdateActivity()
```
