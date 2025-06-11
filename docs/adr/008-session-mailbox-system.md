# 8. Session Mailbox System

Date: 2025-01-11

## Status

Accepted

## Context

AI agents working in isolated sessions need a way to communicate asynchronously with users and potentially other systems. Direct stdin/stdout communication is planned for the future, but we need an immediate solution for:

1. Sending instructions or updates to running agents
2. Receiving status updates or questions from agents
3. Maintaining communication history for context
4. Supporting both programmatic and manual interaction

## Decision

We implemented a file-based mailbox system where each session has a dedicated directory structure for asynchronous message exchange.

### Architecture

#### Directory Structure
```
.amux/mailbox/{session-id}/
├── in/                    # Messages TO the agent
│   └── {timestamp}-{name}.md
├── out/                   # Messages FROM the agent
│   └── {timestamp}-{name}.md
└── context.md            # Instructions for agents
```

#### Message Format
- Files use Unix timestamp prefix for ordering: `1736598935-message-name.md`
- Markdown format for rich content support
- Timestamp prefix enables chronological sorting
- Human-readable names after timestamp

#### Command Structure
We chose a subcommand structure under `mailbox` (alias: `mb`) to:
1. Group related functionality
2. Avoid namespace conflicts with future direct communication commands
3. Make the mailbox concept explicit

Commands follow Unix philosophy with clear separation of concerns:
- `send` - Write to agent's inbox (supports stdin/file/args)
- `recv` - Read latest from agent's outbox
- `show` - Flexible message viewing (by index/latest/all)
- `list` - Directory listing with indices

### Design Decisions

#### 1. File-based over Database
- **Chosen**: Individual files in directories
- **Rationale**: 
  - Transparent and debuggable
  - Works with existing file tools
  - No additional dependencies
  - Easy for agents to understand and use
  - Natural integration with markdown-based workflows

#### 2. Mailbox Metaphor
- **Chosen**: Inbox/outbox directory structure
- **Rationale**:
  - Clear mental model for users
  - Unambiguous direction semantics
  - Familiar concept from email

#### 3. Command Design (Option 3)
After considering multiple options, we chose:
- `send`/`recv` for primary agent communication
- `show` for flexible message inspection
- `list` for overview with indices

**Rejected alternatives**:
- Option 1: `send`/`read` - "read" was ambiguous
- Option 2: `write`/`read` - less intuitive than send/recv

#### 4. Index-based Access
- Messages are numbered globally across in/out directories
- 1-based indexing for user friendliness
- Sorted by timestamp (newest first)
- Enables quick access: `amux mb show s1 3`

#### 5. Automatic Initialization
- Mailbox created automatically when session starts
- Includes context.md with instructions
- No manual setup required

## Consequences

### Positive
- Simple, transparent architecture
- No runtime dependencies
- Easy to debug and inspect
- Natural fit for AI agents that work with files
- Supports both manual and programmatic use
- Unix-friendly with stdin/stdout support
- Clear upgrade path to direct communication

### Negative
- File system overhead for many small messages
- No real-time notification mechanism
- Potential race conditions with concurrent access
- Manual cleanup may be needed for old messages

### Future Considerations
- Could add file watching for real-time updates
- May want message size limits
- Could implement message expiration
- Direct stdin/stdout communication will complement, not replace this system

## Implementation Details

The mailbox system is implemented in `internal/core/mailbox/` with:
- `Manager` type handling all operations
- Integration with session lifecycle
- Commands in `internal/cli/commands/mailbox_*.go`
- Full test coverage

Example usage:
```bash
# Send a message
echo "Focus on auth module" | amux mb send s1

# Get latest from agent
amux mb recv s1

# Read specific message
amux mb list s1
amux mb show s1 3
```