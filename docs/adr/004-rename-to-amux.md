# 4. Rename Project to Amux

Date: 2025-06-10

## Status

Accepted

## Context

This supersedes ADR-002 which established the "Cave" terminology.

The current name "AgentCave" has several issues:

- Too long for a CLI command (9 characters)
- "Cave" metaphor has negative connotations (dark, primitive)
- Doesn't clearly convey the tool's purpose
- Sounds unprofessional for enterprise adoption

With the planned agent multiplexing feature, we need a name that:

- Works well as a CLI command
- Conveys the multiplexing concept
- Scales from single workspace to multi-agent orchestration

## Decision

Rename the project to **Amux** (Agent Multiplexer).

This supersedes the naming decision in ADR-002 which established "AgentCave" and the "Cave" terminology.

## Rationale

**Why "Amux":**

- Follows established pattern (tmux = terminal multiplexer)
- Short and easy to type (4 characters)
- Developers immediately understand "multiplexing"
- Professional and brandable
- Clear pronunciation ("ay-mux")
- Works well as both project name and CLI command

**Problems with "AgentCave":**

- Too long for a CLI command (9 characters)
- "Cave" metaphor has negative connotations
- Doesn't clearly convey the tool's purpose
- Less professional for enterprise adoption

## Consequences

- Need to update all references from AgentCave to Amux
- Binary name changes from `agentcave` to `amux`
- Repository may need renaming
- Documentation updates required
- Much better CLI UX for users
- Clear path for agent multiplexing features
