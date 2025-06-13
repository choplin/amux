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

Key implementation details:

- Added `CapturePaneWithOptions` to tmux adapter with `-e` flag for escape sequences
- Enhanced tail package with refresh mode and configurable options
- Added `--interval` flag to customize refresh rate
- Use FNV-1a hash for fast change detection

## Consequences

### Positive

- Captures all AI agent output including progress updates
- Clean, reliable output without corruption
- Minimal performance impact (60 subprocess calls/minute at 1s interval)
- Preserves colors and formatting
- Configurable for different use cases

### Negative

- Not real-time (1-second delay by default)
- Full screen refresh might cause slight flicker
- Requires terminal that supports ANSI escape codes

### Neutral

- Users needing real-time can use `tmux attach`
- Refresh rate is configurable via `--interval`
- Trade-off of slight delay for reliability is acceptable
