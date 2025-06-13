# 20. Session Status Tracking

Date: 2025-06-13

## Status

Accepted

## Context

Amux manages AI agent sessions running in tmux. Previously, sessions only had basic lifecycle states
(created, stopped, failed) but provided no insight into whether an AI agent was actively working or
waiting for input. Users needed to manually attach to sessions or check output to understand agent
activity.

The challenge is that amux is a one-shot CLI command, not a long-running daemon, so we cannot use
background goroutines for continuous monitoring.

## Decision

Implement on-demand status checking that updates session status only when explicitly requested by
commands. Use hash-based change detection on the last 20 lines of tmux output to determine if a
session is "working" (output changing) or "idle" (no output for 3+ seconds).

Status updates occur when:

- Listing sessions (`amux ps`, `amux status`)
- Sending input to a session
- MCP tools request session information

The status state is persisted in session YAML files to survive across CLI invocations.

## Consequences

This approach provides zero overhead (no background processes) and accurate status detection with
minimal performance impact. However, status is not real-time and only updates when explicitly
requested. This trade-off aligns with amux's CLI architecture while still providing useful activity
information to users.
