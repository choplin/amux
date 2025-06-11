# 7. Table Library Selection

Date: 2025-01-06

## Status

Accepted

## Context

When implementing table formatting for list commands (Issue #8), we needed to choose a Go library for rendering
tabular data in the CLI. Two primary options were considered:

1. **rodaine/table** - A simple, lightweight table formatting library
2. **jedib0t/go-pretty** - A feature-rich formatting library with tables, progress bars, and more

The decision impacts:

- User experience when viewing lists of workspaces and sessions
- Code complexity and maintainability
- Binary size and dependencies
- Future extensibility

## Decision

We will use **rodaine/table** for table formatting in Amux.

## Consequences

### Positive

- **Simplicity**: The API is straightforward and easy to use
- **Lightweight**: Minimal dependencies align with our single-binary distribution goal
- **Sufficient features**: Provides header formatting, column formatting, and padding control
- **Good integration**: Works well with our existing lipgloss styling
- **Already implemented**: No migration cost as it's already integrated in PR #15

### Negative

- **Limited features**: No built-in borders, cell merging, or per-column alignment
- **Basic styles**: Must implement all styling ourselves (though we already use lipgloss)
- **Less active development**: Updates less frequently than go-pretty

### Neutral

- We've created a `NewTable()` wrapper that provides consistent styling across the application
- The simple API means less to learn and maintain

## Alternative Considered: jedib0t/go-pretty

While go-pretty offers more features (borders, multiple output formats, built-in styles, progress bars), these
capabilities are not needed for Amux's focused use case. The additional complexity and larger dependency footprint
don't provide sufficient value for a CLI tool that prioritizes simplicity.

## Implementation

The table functionality is wrapped in `internal/cli/ui/table.go`:

```go
func NewTable(headers ...interface{}) table.Table {
    tbl := table.New(headers...)
    tbl.WithHeaderFormatter(headerStyle)
    tbl.WithFirstColumnFormatter(boldStyle)
    tbl.WithPadding(2)
    return tbl
}
```

This provides consistent styling across all tables in the application while keeping the implementation simple.
