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

Agents are configured in `.amux/config.yaml`. To view or modify agent configurations:

```bash
# View current configuration
amux config show

# Edit configuration
amux config edit
```

Example agent configuration:

```yaml
agents:
  claude:
    name: Claude
    type: tmux
    environment:
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
    params:
      command: claude code
      shell: /bin/zsh        # Optional: custom shell
      windowName: claude-dev # Optional: tmux window name
```

### Tmux Parameters

Agents support optional tmux parameters:

- **shell**: Custom shell for the session (e.g., `/bin/zsh`, `/bin/fish`)
- **windowName**: Custom name for the tmux window
- **autoAttach**: Automatically attach to session when run from CLI (default: false)
- **environment**: Environment variables (can also be set at runtime via MCP)

The `autoAttach` parameter is particularly useful for interactive debugging or when you need immediate access to the session. When enabled and running from a terminal, Amux will automatically attach to the tmux session after creation.
