# 19. Log Following Refresh Approach

Date: 2025-06-13

## Status

Accepted

## Context

The `amux session logs -f` feature was not capturing AI agent output correctly. AI agents often use carriage
returns (\r) for progress bars, spinners, and other dynamic terminal updates. The tmux `capture-pane` command
only shows the final terminal state, missing intermediate updates.

Initial investigation showed:

- Standard `capture-pane` misses intermediate states when agents use \r for progress updates
- The `pipe-pane` approach captures all output but produces corrupted, unusable results
- Users need to monitor AI agent progress without attaching to the session directly

## Decision

We implemented a refresh-based approach using `tmux capture-pane` with smart optimizations:

1. **Periodic refresh** instead of continuous streaming (default 1 second, configurable)
2. **Terminal-height capture** to limit data transfer
3. **Hash-based change detection** to skip unnecessary redraws
4. **Full screen clear** before each redraw for clean rendering
5. **ANSI escape sequence preservation** for colored output

### Performance Optimization Journey

During implementation, we discovered a critical performance issue: the initial implementation was capturing
the ENTIRE tmux buffer (potentially thousands of lines) every second, then trimming it to terminal size.
This was fixed by:

1. Modifying the `Session` interface to accept a line limit
2. Calculating terminal size dynamically before each capture
3. Passing this limit down to tmux to capture ONLY what we display
4. Removing post-processing since we capture exactly what we need

This resulted in a 50-100x reduction in data transfer for long-running sessions.

Key implementation details:

- Added `CapturePaneWithOptions` to tmux adapter with `-e` flag for escape sequences
- Modified `Session.GetOutput()` interface to accept line limit parameter
- Dynamic terminal size detection before each capture
- Capture ONLY terminal-visible lines from tmux (e.g., 40 lines vs 2000+)
- Use Go's standard `hash/fnv` for fast change detection
- Added `--interval` flag to customize refresh rate (default 1s)
- Removed unnecessary `processOutput()` function since we capture exactly what we need

## Consequences

### Positive

- Captures all AI agent output including progress updates
- Clean, reliable output without corruption
- **Dramatic performance improvement**: Only captures terminal-visible lines (50-100x reduction)
- Minimal data transfer: ~40 lines instead of entire buffer (2000+ lines)
- Preserves colors and formatting with `-e` flag
- Automatically handles terminal resize
- Configurable refresh rate for different use cases

### Negative

- Not real-time (1-second delay by default)
- Full screen refresh might cause slight flicker
- Requires terminal that supports ANSI escape codes

### Neutral

- Users needing real-time can use `tmux attach`
- Refresh rate is configurable via `--interval`
- Trade-off of slight delay for reliability is acceptable
