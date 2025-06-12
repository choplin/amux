# ADR-017: Synchronous Session Communication

## Status

Accepted

## Context

Currently, amux uses synchronous function calls to communicate with agent sessions through the tmux adapter.
Each operation (`SendInput`, `GetOutput`, `Start`, `Stop`, etc.) is a blocking call that executes a tmux
command and returns the result.

We considered migrating to a goroutine/channel-based asynchronous communication model to potentially handle:

- Concerns about losing input/output between calls
- More real-time interaction patterns
- Multiple concurrent consumers of session data

## Decision

We will maintain the current synchronous communication model for session management.

## Rationale

### Advantages of Current Approach

1. **Tmux as the Buffer Layer**
   - Tmux already provides robust input buffering and queuing
   - Terminal emulation and output history are handled by tmux
   - No risk of losing input - tmux queues it properly

2. **Simplicity**
   - Stateless operations are easier to reason about
   - No goroutine lifecycle management
   - No risk of goroutine leaks or channel deadlocks
   - Easier to test and debug

3. **Appropriate for Use Case**
   - AI agents don't require microsecond-level response times
   - Command-line tools naturally fit synchronous patterns
   - Current polling approach is sufficient for monitoring

4. **Maintainability**
   - Less complex codebase
   - Clear error propagation
   - Predictable behavior

### When Async Would Be Needed

We would reconsider this decision if we need:

- Real-time output streaming (e.g., for `amux tail` implementation)
- High-frequency input/output operations
- Multiple concurrent consumers of session events
- WebSocket or gRPC streaming APIs

## Consequences

### Positive

- Simpler codebase to maintain
- Easier onboarding for contributors
- More reliable and predictable behavior
- Lower cognitive overhead

### Negative

- Potential output gaps between `GetOutput` calls (acceptable for current use cases)
- Slight performance overhead from spawning tmux processes (negligible in practice)
- Would need refactoring if real-time features are required

## Implementation Notes

- Keep the current `Session` interface with synchronous methods
- Use goroutines only for specific features that require them (e.g., future `amux tail`)
- Consider buffered output collection as a focused enhancement when needed
- Document that output capture is point-in-time, not continuous

## References

- Issue #6: Log tailing feature (would benefit from async streaming)
- Session interface: `internal/core/session/types.go`
- Tmux adapter: `internal/adapters/tmux/adapter.go`
