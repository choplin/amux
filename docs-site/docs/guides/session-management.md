---
sidebar_position: 2
---

# Session Management

Monitor and control AI agent sessions running in Amux workspaces.

## Overview

Sessions are running instances of configured agents. Each session:

- Runs in a tmux session
- Executes within a specific workspace
- Operates independently from other sessions
- Displays real-time status (running, idle, stuck)

## Running Agents

### Quick Start

```bash
# Run default agent
amux run claude

# Run in specific workspace
amux run claude --workspace feature-auth

# Named session
amux run aider --name "code-review"
```

### Parallel Agents

```bash
# Start multiple agents
amux run claude --workspace feat-1 &
amux run aider --workspace feat-2 &
amux run my-assistant --workspace feat-3 &

# Monitor all
amux ps
```

## Managing Sessions

### List Sessions

```bash
amux ps          # Running sessions
amux ps --all    # All sessions
```

### Attach to Session

```bash
amux attach session-123
# Detach with Ctrl+B, D
```

### View Logs

```bash
amux logs session-123
amux tail session-123  # Follow logs in real-time
```

## Agent Configuration

### List Available Agents

```bash
amux agent list
```

### Show Agent Details

```bash
amux agent show claude
```

Agents are configured in `.amux/config.yaml`. To modify agents:

```bash
amux config edit
```
