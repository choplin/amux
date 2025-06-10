# 2. Cave Terminology and Concept

Date: 2025-06-08

## Status

Accepted

## Context

When starting this project, we needed a clear metaphor for isolated development environments where AI agents work
autonomously. The name and concept should convey both isolation and purpose.

## Decision

- Name the project "AgentCave"
- Define "Cave" as an isolated development environment for AI agents
- Focus on workspace provision rather than centralized orchestration

## Rationale

**Cave as a Metaphor**:

- Caves are isolated, private spaces
- Suggests a place where work happens independently
- More memorable and distinctive than generic terms

**Cave Definition**:
A "Cave" is an isolated development environment consisting of:

1. A git worktree (physical isolation)
2. Working Context files (cognitive context)
3. Private workspace for an AI agent

**Working Context Files**:

- `background.md` - Requirements and constraints
- `plan.md` - Implementation approach
- `working-log.md` - Real-time progress tracking
- `results-summary.md` - Final outcomes

## Consequences

- Clear project identity from the start
- Strong metaphor for isolated workspaces
- Sets expectation of agent autonomy over central control
- Need to integrate Working Context templates to fulfill the "Cave" concept
- Documentation uses "cave" and "workspace" somewhat interchangeably
