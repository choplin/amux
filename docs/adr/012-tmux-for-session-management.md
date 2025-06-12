# ADR-012: Tmux for Session Management

Date: 2025-06-12

## Status

Accepted

## Context

Amux needs to manage long-running terminal sessions for AI agents. These sessions must:

- Persist beyond the initial command execution
- Allow attachment and detachment
- Support sending commands programmatically
- Capture output for monitoring
- Work across SSH connections
- Be discoverable and manageable

Options considered:

1. **GNU Screen** - Alternative terminal multiplexer
2. **Custom PTY management** - Direct pseudo-terminal handling
3. **Docker containers** - Container-based sessions
4. **Tmux** - Terminal multiplexer
5. **Simple background processes** - Basic process management

## Decision

We will use Tmux as the primary backend for session management, with a fallback to basic sessions when Tmux is unavailable.

The implementation will:

- Use tmux for persistent terminal sessions
- Provide a clean adapter interface for future backends
- Support basic sessions as a fallback
- Manage tmux sessions with amux-specific naming

## Consequences

### Positive

- **Persistent sessions** - Sessions survive network disconnections
- **Attach/detach capability** - Users can connect to running sessions
- **Standard tool** - Tmux is widely available and well-documented
- **Rich feature set** - Window management, scripting, output capture
- **SSH-friendly** - Works well over remote connections
- **Programmatic control** - Full API for session manipulation

### Negative

- **External dependency** - Requires tmux installation
- **Platform variations** - Behavior differences across OS versions
- **Learning curve** - Users need basic tmux knowledge
- **Resource overhead** - Each session has tmux process overhead

### Neutral

- Tmux configuration affects behavior
- Users can use native tmux commands alongside amux
- Session naming must avoid conflicts

## Implementation Notes

1. Sessions are named `amux-{workspace}-{id}` for easy identification
2. The `TmuxAdapter` implements the `Adapter` interface
3. Environment variables are set at session creation time
4. Fallback to `BasicSession` when tmux is unavailable
5. Session metadata is stored separately from tmux state

## Alternatives Considered

### GNU Screen

**Pros**: Similar features to tmux, mature tool
**Cons**: Less active development, fewer features, less programmatic control

### Custom PTY Management

**Pros**: No external dependencies, full control
**Cons**: Complex implementation, platform-specific code, reinventing the wheel

### Docker Containers

**Pros**: Strong isolation, reproducible
**Cons**: Heavy dependencies, complex networking, requires Docker daemon

### Simple Background Processes

**Pros**: Simple implementation, no dependencies
**Cons**: No persistence, no attach capability, limited control

## Future Considerations

The adapter pattern allows for future backends:

- Docker/Kubernetes for cloud environments
- Custom PTY for embedded systems
- Remote session protocols for distributed setups

## References

- [Tmux Manual](https://man7.org/linux/man-pages/man1/tmux.1.html)
- [Terminal Multiplexers Comparison](https://en.wikipedia.org/wiki/Terminal_multiplexer)
- Session Adapter Interface in `internal/adapters/tmux/adapter.go`
