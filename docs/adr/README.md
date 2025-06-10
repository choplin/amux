# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the AgentCave project.

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
