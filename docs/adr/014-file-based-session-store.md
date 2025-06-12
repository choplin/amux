# ADR-014: File-based Session Store

Date: 2025-06-12

## Status

Accepted

## Context

Amux needs to persist session metadata across restarts. This includes:

- Session ID and status
- Associated workspace
- Agent configuration
- Start time and other metadata

Requirements:

- Simple implementation
- No external dependencies
- Easy debugging and inspection
- Portable across systems
- Consistent with other amux storage

Options considered:

1. **SQLite database** - Embedded SQL database
2. **BoltDB/BadgerDB** - Embedded key-value stores
3. **JSON files** - One file per session
4. **YAML files** - One file per session (consistent with other storage)
5. **Memory-only** - No persistence

## Decision

We will use a file-based session store with individual YAML files for each session.

Implementation:

- Store session files in `.amux/sessions/`
- One YAML file per session: `session-{id}.yaml`
- In-memory cache for performance
- File system as source of truth

## Consequences

### Positive

- **Simple implementation** - Just file I/O operations
- **No dependencies** - No database libraries needed
- **Easy debugging** - Can inspect/edit files directly
- **Portable** - Works on any filesystem
- **Consistent** - Same format as other amux data
- **Recovery-friendly** - Can manually fix corrupted data
- **Version control friendly** - Changes are trackable

### Negative

- **Race conditions** - Potential conflicts with concurrent access
- **Limited querying** - No SQL-like queries, must scan files
- **Performance at scale** - Listing many sessions requires reading many files
- **No transactions** - No ACID guarantees
- **Manual cleanup** - Orphaned files need detection

### Neutral

- File system performance characteristics apply
- Backup/restore is just file copying
- Migration requires file format changes

## Implementation Notes

1. Session files are named `session-{id}.yaml`
2. Use file locking for write operations
3. Implement in-memory cache with TTL
4. Scan directory for listing operations
5. Use atomic writes (write to temp, rename)

Example session file:

```yaml
id: sess-abc123
workspace_id: ws-xyz789
agent_id: claude
status: running
tmux_session: amux-myworkspace-abc123
created_at: 2024-01-15T10:30:00Z
updated_at: 2024-01-15T10:35:00Z
environment:
  ANTHROPIC_API_KEY: "***"
```

## Alternatives Considered

### SQLite Database

**Pros**: ACID transactions, SQL queries, indexes, single file
**Cons**: Dependency, migration complexity, debugging harder, overkill

### BoltDB/BadgerDB

**Pros**: Fast key-value operations, transactions, Go-native
**Cons**: Binary format, debugging difficult, dependencies

### JSON Files

**Pros**: Simple, wide support
**Cons**: No comments, less readable than YAML

### Memory-only

**Pros**: Fast, simple
**Cons**: No persistence, sessions lost on restart

## Future Migration Path

If file-based storage becomes inadequate:

1. Implement new store interface
2. Add migration command
3. Support both stores temporarily
4. Gradually migrate sessions
5. Remove file-based implementation

The `SessionStore` interface makes this migration straightforward.

## Performance Considerations

- Cache frequently accessed sessions
- Implement pagination for list operations
- Consider sharding by date for many sessions
- Monitor file system limits

## References

- File-based storage patterns
- Go file locking best practices
- [ADR-013: YAML for Configuration](013-yaml-for-configuration.md)
