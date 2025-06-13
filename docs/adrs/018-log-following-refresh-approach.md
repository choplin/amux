# ADR-018: Log Following Refresh Approach

**Status**: Accepted
**Date**: 2025-01-13
**Deciders**: @choplin, @claude

## Context

The `amux session logs -f` feature was not capturing AI agent output correctly. AI agents often use carriage
returns (\r) for progress bars, spinners, and other dynamic terminal updates. The tmux `capture-pane` command
only shows the final terminal state, missing intermediate updates.

Initial investigation showed:

- Standard `capture-pane` misses intermediate states when agents use \r for progress updates
- The `pipe-pane` approach captures all output but produces corrupted, unusable results
- Users need to monitor AI agent progress without attaching to the session directly

## Decision

We implemented a **refresh-based approach** using `tmux capture-pane` with smart optimizations:

1. **Periodic refresh** instead of continuous streaming (default 1 second, configurable)
2. **Terminal-height capture** to limit data transfer
3. **Hash-based change detection** to skip unnecessary redraws
4. **Full screen clear** before each redraw for clean rendering
5. **ANSI escape sequence preservation** for colored output

## Rationale

### Why not pipe-pane?

- Testing showed pipe-pane produces "wrecked output" that is "completely useless"
- The output corruption makes it unsuitable for production use

### Why refresh approach?

- Simple and reliable - uses well-tested tmux capture functionality
- Clean output without corruption issues
- Acceptable trade-off: slight delay for reliability
- Users can still `tmux attach` for real-time viewing when needed

### Performance optimizations

- **FNV-1a hash**: O(n) operation but very fast for change detection
- **Limited capture**: Only terminal height (~30-80 lines) instead of full buffer
- **Skip unchanged**: No redraw if content hasn't changed
- **1-second default**: 60 subprocess calls/minute is minimal load

## Implementation Details

### Key components

1. **CapturePaneWithOptions** in tmux adapter:

   ```go
   func (a *RealAdapter) CapturePaneWithOptions(sessionName string, lines int) (string, error) {
       args := []string{"capture-pane", "-t", sessionName, "-p", "-J", "-e"}
       if lines > 0 {
           args = append(args, "-S", fmt.Sprintf("-%d", lines))
       }
       // ...
   }
   ```

2. **Tail package** with refresh mode:
   - Configurable poll interval
   - Terminal size detection
   - Hash-based change detection
   - Full screen clear with ANSI codes

3. **CLI integration**:
   - `--interval` flag for custom refresh rates
   - Clean status messages
   - Graceful signal handling

### ANSI color preservation

- Must use `-e` flag with `tmux capture-pane`
- Pass through escape sequences unchanged
- Let user's terminal handle rendering

## Consequences

### Positive

- ✅ Captures all AI agent output including progress updates
- ✅ Clean, reliable output without corruption
- ✅ Minimal performance impact
- ✅ Preserves colors and formatting
- ✅ Configurable for different use cases

### Negative

- ❌ Not real-time (1-second delay by default)
- ❌ Full screen refresh might cause slight flicker
- ❌ Requires terminal that supports ANSI escape codes

### Neutral

- Users needing real-time can use `tmux attach`
- Refresh rate is configurable via `--interval`

## References

- Issue: #111 (Fix logs -f to capture AI agent output)
- tmux capture-pane documentation
- ANSI escape code standards
