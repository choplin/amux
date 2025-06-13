# ADR-018: Separate Agent and Session Commands

## Status

Accepted

## Context

The original command structure had all session management operations under the `agent` command:
- `amux agent run` - Run a session
- `amux agent list` - List running sessions
- `amux agent attach` - Attach to a session
- `amux agent stop` - Stop a session

This created confusion because:
1. **Agent** represents a static configuration (which AI tool to run, default commands, environment)
2. **Session** represents a running process instance created from an agent configuration

The mismatch between the command name and its actual function made the CLI less intuitive.

## Decision

We will separate agent and session commands to match their conceptual roles:

### Session Commands (Managing Running Processes)
Create a new `session` command for all runtime operations:
- `amux session run <agent>` - Run an agent session
- `amux session list` - List active sessions
- `amux session attach <session>` - Attach to a session
- `amux session stop <session>` - Stop a session
- `amux session logs <session>` - View session output

### Agent Commands (Viewing Configurations)
Simplify `agent` command to be read-only:
- `amux agent list` - List configured agents
- `amux agent show <agent>` - Show agent configuration details

Configuration changes will be handled through `amux config edit` (see ADR-019).

### Global Aliases
Maintain backward compatibility with common shortcuts:
- `amux run` → `amux session run`
- `amux ps` → `amux session list`
- `amux attach` → `amux session attach`
- `amux tail` → `amux session logs -f`

## Rationale

### Clarity and Intuition
- Commands now match their actual purpose
- Users can understand functionality from command names
- Clear mental model: agents are templates, sessions are instances

### Consistency
- Follows common CLI patterns (e.g., Docker's image vs container)
- Separation of configuration from runtime aligns with Unix philosophy

### Extensibility
- Easy to add session-specific features (send-input, restart, etc.)
- Agent commands can focus on configuration inspection
- Future config management features won't conflict with runtime operations

### User Experience
- Global aliases preserve muscle memory for common operations
- Help text clearly explains the distinction
- Error messages can be more specific

## Consequences

### Positive
- Clearer command structure improves discoverability
- Reduced cognitive load for new users
- Better foundation for future features
- Maintains backward compatibility through aliases

### Negative
- Breaking change for scripts using full command paths
- Users need to learn new command structure
- Documentation and examples need updating

### Migration
- Existing users can continue using global aliases
- Full command paths in scripts need updating
- Help text guides users to new structure

## Implementation Notes

- Move all session-related code to `internal/cli/commands/session/`
- Keep agent commands in `internal/cli/commands/agent/` but simplify
- Update root command to register both command sets
- Ensure aliases work correctly with flag inheritance

## Future Considerations

- `amux session send-input` for programmatic interaction (#97)
- `amux config` for configuration management (#101)
- Potential session groups or named session management
- Session templates for common configurations

## References

- Issue #100: Command structure refactoring
- Issue #101: Config command implementation
- Issue #97: Send-input functionality
- PR #104: Implementation of this ADR
