# 22. Session Failure Detection

Date: 2025-06-15

## Status

Accepted

## Context

Currently, `StatusFailed` is defined but never set in the code. We need to detect and properly set
the failed status to provide better visibility into session health. Additionally, we identified a gap
where sessions can complete successfully but remain open, which isn't represented in our state model.

For tmux-based sessions, we have three objects to consider:

- Tmux session (open/closed)
- Shell process (running/exited)
- Command process (running/finished successfully/finished with error)

## Decision

Add `StatusCompleted` as a new state and implement failure detection based on process states:

### State Mapping

| Tmux Session | Shell Process | Child Processes | Exit Status | Session Status |
|--------------|---------------|-----------------|-------------|----------------|
| Closed | - | - | - | `StatusFailed` (error: "tmux session no longer exists") |
| Open | Dead | - | - | `StatusFailed` (error: "shell process exited") |
| Open | Running | No children | 0 | `StatusCompleted` |
| Open | Running | No children | Non-zero | `StatusFailed` (error: "command exited with code N") |
| Open | Running | Has children | - | `StatusWorking` (if output changing) |
| Open | Running | Has children | - | `StatusIdle` (if no output change for 3s) |

### Implementation

1. Add `StatusCompleted` to represent successfully finished commands
2. In `UpdateStatus()` check in order:
   - If tmux session doesn't exist → `StatusFailed`
   - If shell process is dead (using `pane_dead`) → `StatusFailed`
   - If shell has no child processes → Check for exit status
   - Otherwise continue with existing working/idle detection

### Performance Optimizations

To improve performance when listing many sessions:

1. **Status Caching**: Added 2-second cache to `UpdateStatus` to avoid redundant checks
2. **Batch Updates**: Implemented `UpdateAllStatuses` for parallel status updates
3. **Efficient Session List**: Uses batch updates in `amux ps` and MCP resources

### Exit Status Tracking

Implemented automatic exit status capture:

- When no child processes remain, we send `echo $? > {storage}/exit_status` to the shell
- This writes the exit code directly to a file, avoiding shell prompt parsing
- Read the exit status from the file after a 100ms delay
- Exit code 0 → `StatusCompleted`
- Non-zero exit code → `StatusFailed` with "command exited with code N" error
- If exit status capture fails, the session still transitions to `StatusCompleted` (assumes success)

This provides robust exit status tracking without depending on shell prompt format.

### Update Behavior

The `UpdateStatus()` method has specific behaviors:

- Only runs when session is in a running state (`StatusWorking` or `StatusIdle`)
- Once a session reaches a terminal state (`StatusCompleted`, `StatusStopped`, `StatusFailed`), no further updates occur
- Uses a 2-second cache to prevent excessive external process calls
- Updates are thread-safe using mutex locking

### Idle Detection

Sessions transition from `StatusWorking` to `StatusIdle` when:

- The session has child processes running
- No output change is detected for 3 seconds (idleThreshold)
- Output comparison uses FNV-1a hash for efficiency

We will not attempt to:

- Distinguish between different types of failures (crash vs error)
- Detect command launch failures (too heuristic)
- Parse shell output for exit codes (fragile and shell-dependent)

### Platform-Specific Considerations

1. **Windows Support**: Tests that require tmux or pgrep are skipped on Windows
2. **Process Detection**: Uses platform-specific process checking (pgrep on Unix-like systems)

### Session Cleanup on Removal

When removing a session (via `amux session remove`), any remaining tmux session is also cleaned up:

- This ensures no orphaned tmux sessions are left behind
- Applies to all terminal states: `completed`, `stopped`, and `failed`
- Prevents accumulation of unused tmux sessions after session removal
- The cleanup happens automatically in `Manager.Remove()` method

### Session Lifecycle and State Persistence

Once a session reaches a terminal state (`completed`, `stopped`, or `failed`), its status is preserved:

- `StatusCompleted`: Command finished successfully (exit code 0)
- `StatusFailed`: Command failed (non-zero exit code) or session crashed
- `StatusStopped`: User explicitly stopped the session

The status remains unchanged even if the underlying tmux session is closed. This design choice:

- Preserves important information about command execution results
- Simplifies state management by avoiding additional state transitions
- Allows users to see the final outcome of their sessions in history

## Consequences

This approach provides clear session state visibility without complex heuristics. Users can see when
their agents have completed work versus failed. The implementation remains simple by relying on
process hierarchy rather than trying to parse output or track exit codes.

The performance optimizations ensure that the status updates don't impact the responsiveness of
list operations, even with many sessions.
