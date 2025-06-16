---
sidebar_position: 1
---

# Welcome to Amux

## The Challenge

Modern development workflows often require:

- Switching between multiple features and bug fixes
- Running AI assistants on different parts of the codebase
- Maintaining context across different tasks
- Avoiding conflicts between parallel work streams

Traditional Git workflows force constant stashing, branch switching, and context loss. Running multiple AI agents means dealing with conflicting changes and confused assistants. The result: reduced productivity and increased frustration.

## What is Amux

**Amux** (Agent Multiplexer) is a workspace management tool that creates isolated Git worktree-based environments. Key features:

- **Parallel AI agent execution** - Run multiple agents without conflicts
- **Instant context switching** - Move between tasks without stashing
- **Complete isolation** - Each workspace has its own branch and files
- **Fast workspace creation** - New environments in seconds

## See It In Action

```bash
# Run AI agents with auto-created workspaces
amux run claude feat-auth
amux run aider fix-bug-123
amux run claude docs-update

# Monitor all sessions
amux ps
┌─────────┬────────┬──────────────┬────────┐
│ SESSION │ AGENT  │ WORKSPACE    │ STATUS │
├─────────┼────────┼──────────────┼────────┤
│ 1       │ claude │ feat-auth    │ 🟢 busy │
│ 2       │ aider  │ fix-bug-123  │ 🟢 busy │
│ 3       │ claude │ docs-update  │ 🟢 idle │
└─────────┴────────┴──────────────┴────────┘
```

## Use Cases

### For AI-Assisted Development

- Run multiple AI agents on different features simultaneously
- Each agent maintains its own context and workspace
- Prevent conflicts between different AI assistants

### For Human Developers

- Move between features and bug fixes without stashing
- Keep experimental changes isolated
- Maintain multiple work streams in parallel

### For Teams

- Provide isolated environments for new developers
- Test changes without affecting the main branch
- Enable truly parallel development workflows

## Built on Proven Technology

Amux leverages battle-tested tools that developers already trust:

### Git Worktrees

- **Standard Git feature** - No proprietary formats or magic
- **Full compatibility** - Works with your existing Git workflow
- **Direct access** - Workspaces are just directories you can navigate

### Tmux Sessions

- **Reliable terminal multiplexer** - Industry-standard session management
- **Persistent sessions** - AI agents keep running even if disconnected
- **Native terminal experience** - No custom protocols or interfaces

## Getting Started

```bash
# Install (macOS/Linux)
brew install choplin/amux/amux

# Initialize your project
amux init

# Create your first workspace
amux ws create my-feature

# Start developing!
```

## Next Steps

- **[Installation](getting-started/installation)** - Get Amux running in under a minute
- **[Quick Start](getting-started/quick-start)** - Create your first workspace
- **[AI Workflows](guides/ai-workflows)** - Set up MCP for Claude Code
