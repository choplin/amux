# 22. Session Failure Detection

Date: 2025-06-14

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

| Tmux Session | Shell | Command | Session Status |
|--------------|-------|---------|----------------|
| Closed | - | - | `StatusFailed` |
| Open | Exited | - | `StatusFailed` |
| Open | Running | No children | `StatusCompleted` |
| Open | Running | Has children + output changing | `StatusWorking` |
| Open | Running | Has children + no output change | `StatusIdle` |

### Implementation

1. Add `StatusCompleted` to represent successfully finished commands
2. In `UpdateStatus()` check in order:
   - If tmux session doesn't exist → `StatusFailed`
   - If shell process is dead (using `pane_dead`) → `StatusFailed`
   - If shell has no child processes → `StatusCompleted`
   - Otherwise continue with existing working/idle detection

### Exit Status Tracking

Implemented automatic exit status capture:

- When no child processes remain, we send `echo $? > {storage}/exit_status` to the shell
- This writes the exit code directly to a file, avoiding shell prompt parsing
- Read the exit status from the file
- Exit code 0 → `StatusCompleted`
- Non-zero exit code → `StatusFailed` with "command exited with code N" error

This provides robust exit status tracking without depending on shell prompt format.

We will not attempt to:

- Distinguish between different types of failures (crash vs error)
- Detect command launch failures (too heuristic)
- Parse shell output for exit codes (fragile and shell-dependent)

## Consequences

This approach provides clear session state visibility without complex heuristics. Users can see when
their agents have completed work versus failed. The implementation remains simple by relying on
process hierarchy rather than trying to parse output or track exit codes.
