# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the Amux project.

## Index

1. [Record Architecture Decisions](001-record-architecture-decisions.md)
2. [Cave Terminology](002-cave-terminology.md)
3. [Agent Multiplexing Architecture](003-agent-multiplexing-architecture.md)
4. [Rename to Amux](004-rename-to-amux.md)
5. [Command Structure](005-command-structure.md)
6. [Context Management Strategy](006-context-management-strategy.md)
7. [Table Library Selection](007-table-library-selection.md)
8. [Session Mailbox System](008-session-mailbox-system.md)
9. [MCP Resources and Prompts](009-mcp-resources-and-prompts.md)
10. [Documentation Structure Strategy](010-documentation-structure.md)
11. [Git Worktrees for Workspace Isolation](011-git-worktrees-for-workspace-isolation.md)
12. [Tmux for Session Management](012-tmux-for-session-management.md)
13. [YAML for Configuration](013-yaml-for-configuration.md)
14. [File-based Session Store](014-file-based-session-store.md)

## What is an ADR?

An Architecture Decision Record captures an important architectural decision made along with its context and
consequences. They help future maintainers understand not just what decisions were made, but why they were made.

## Template

When creating a new ADR, use this template:

```markdown
# N. Title

Date: YYYY-MM-DD

## Status

Accepted

## Context

What is the issue that we're seeing that is motivating this decision?

## Decision

What is the change that we're proposing and/or doing?

## Consequences

What becomes easier or more difficult to do because of this change?
```

## Notes

- Once accepted, ADRs are immutable - they represent historical decisions
- If a decision needs to be changed, create a NEW ADR that supersedes the old one
- The new ADR should reference what it supersedes (e.g., "This supersedes ADR-002")
- Old ADRs remain unchanged to preserve decision history

- The filename format `NNN-descriptive-name.md` helps with ordering and search
