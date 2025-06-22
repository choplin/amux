# 30. Automatic Index Reconciliation

Date: 2025-06-22

## Status

Accepted

## Context

The index state file at `.amux/index/state.yaml` tracks numeric ID mappings for workspaces and sessions. This file can become out of sync with actual entities on disk when:

- Workspaces or sessions are deleted externally (e.g., manual directory deletion)
- File system corruption occurs
- Concurrent operations fail partway through
- Development/debugging activities modify the file system

Currently, there's no mechanism to detect or repair these inconsistencies. Users would need to manually edit `state.yaml` or face confusing behavior where indices point to non-existent entities.

## Decision

Implement automatic index reconciliation at strategic points in the application lifecycle:

1. **During entity listing** - When listing workspaces or sessions, validate all indexed entries and remove orphaned ones
2. **During entity access** - When accessing a specific entity fails, check if it has an orphaned index entry and clean it up
3. **On MCP server startup** - Perform initial reconciliation to ensure clean state for AI agent operations

The reconciliation is implemented by:

- Adding a `Reconcile(entityType, existingIDs)` method to the index manager
- Calling this method with the list of actually existing entities
- Silently removing any index entries that don't correspond to existing entities
- Returning the count of cleaned entries for potential logging

## Consequences

### Positive

- **Self-healing system**: Index inconsistencies are automatically detected and fixed
- **No user intervention needed**: Users don't need to know about or manage index state
- **Transparent operation**: Reconciliation happens during normal operations without user awareness
- **Performance impact minimal**: Only scans index entries, which are typically small in number
- **No new commands**: Keeps the CLI simple without adding maintenance commands

### Negative

- **Silent data modification**: The system modifies state without explicit user consent
- **No audit trail**: Orphaned entries are removed without logging by default
- **Potential race conditions**: Concurrent operations might interfere with reconciliation
- **Hidden complexity**: The automatic behavior might surprise users debugging issues

### Implementation Notes

- Reconciliation only removes definitively orphaned entries (entity doesn't exist on disk)
- Errors during reconciliation don't fail the primary operation
- The existing file locking mechanism prevents concurrent modification issues
- Future enhancement could add debug logging for reconciliation activities
