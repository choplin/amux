---
sidebar_position: 2
---

# Quick Start

Get started with Amux in minutes.

## Workspace Basics

### 1. Initialize Amux

Navigate to your project and initialize Amux:

```bash
cd your-project
amux init
```

This creates a `.amux/` directory to store workspace metadata.

### 2. Create a Workspace

```bash
amux ws create feature-auth
```

You now have an isolated Git worktree ready for development!

### 3. List Your Workspaces

```bash
amux ws list
```

Output:

```text
ID  NAME          BRANCH                           CREATED
1   feature-auth  amux/workspace-feature-auth-...  2 minutes ago
```

## Working with AI Agents

If you have AI agents configured, you can run them in your workspaces:

### 4. Run an AI Agent

```bash
amux run claude --workspace feature-auth
```

### 5. List Sessions

Check your running AI agents:

```bash
amux ps
```

Output:

```text
ID  NAME     AGENT   WORKSPACE     STATUS
1   sess-1   claude  feature-auth  🟢 running
```

### 6. Attach to AI Agent

Connect to your running AI agent for interactive work:

```bash
amux attach 1  # Use session ID from 'amux ps'
```

Exit the session with `Ctrl+B` then `D` (detach) or `exit` (terminate).

## What You've Achieved

### Workspace Setup

✅ Initialized Amux in your project
✅ Created an isolated workspace for feature development
✅ Learned how to manage multiple workspaces

### AI Agent Integration (if completed)

✅ Started an AI agent in an isolated environment
✅ Monitored running sessions
✅ Connected to your AI agent for interactive work

You're now ready to run multiple AI agents in parallel without conflicts!

## Next Steps

- [Workspace Management](../guides/workspaces.md) - Detailed workspace operations
- [Session Management](../guides/session-management.md) - Monitor and control running sessions
- [AI Workflows](../guides/ai-workflows.md) - MCP integration guide
