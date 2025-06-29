<div align="center">
  <h1>
    <img src="assets/logo.svg" alt="Amux" height="32" style="vertical-align: middle">
    Amux
  </h1>
</div>

<div align="center">

![Amux Hero Image](assets/hero-image.svg)

</div>

[![CI](https://github.com/choplin/amux/actions/workflows/ci.yml/badge.svg)](https://github.com/choplin/amux/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/choplin/amux)](https://goreportcard.com/report/github.com/choplin/amux)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/choplin/amux)](https://go.dev/)

> **Isolated workspaces in seconds. Run multiple AI agents without conflicts.**

## Why Amux?

- **Instant Isolation** - Create workspaces in seconds, not minutes
- **True Parallel Development** - Multiple AI agents working without stepping on each other
- **Seamless Integration** - Works naturally with both CLI tools (fzf, ripgrep) and AI agents (via MCP)

## Quick Start

```bash
# 1. Install Amux
brew install choplin/amux/amux

# 2. Initialize your project
cd your-project
amux init

# 3. Create your first workspace
amux ws create feature-auth

# 4. Run an AI agent
amux run claude --workspace feature-auth

# 5. Check running sessions
amux ps
```

That's it! You now have an isolated workspace with an AI agent working on your feature.

## Key Features

### Workspace Management

Create isolated Git worktree environments instantly:

```bash
amux ws create feature-api    # New workspace with new branch
amux ws list                  # See all workspaces
amux ws cd feature-api        # Enter workspace in subshell
```

### Agent Orchestration

Run multiple AI agents in parallel:

```bash
amux run claude --workspace feat-1
amux run gpt --workspace fix-2
amux ps  # Monitor all agents
```

### Built for Your Workflow

#### CLI Integration

```bash
# Interactive selection with fzf
amux ws list | fzf | xargs amux ws cd

# Automation with JSON output
amux ws list --json | jq '.[] | select(.age > 7)'
```

#### AI Agent Integration

Configure in Claude Code:

```json
{
  "mcpServers": {
    "amux": {
      "command": "/usr/local/bin/amux",
      "args": ["mcp", "--git-root", "/path/to/your/project"]
    }
  }
}
```

Then use MCP tools:

```javascript
workspace_create({ name: "feature-auth" })
session_run({ agent_id: "claude", workspace_identifier: "1" })
```

## Installation

### Homebrew (Recommended)

```bash
brew tap choplin/amux
brew install amux
```

### From Source

```bash
git clone https://github.com/choplin/amux.git
cd amux
just build  # or: go build -o bin/amux cmd/amux/main.go
```

### Binary Releases

Download from [releases page](https://github.com/choplin/amux/releases).

## Documentation

Visit **[amux.dev](https://amux.dev)** for:

- 📖 [Complete Guide](https://amux.dev/docs/intro)
- 🛠️ [Command Reference](https://amux.dev/docs/reference/commands)
- 🤝 [MCP Integration](https://amux.dev/docs/guides/ai-workflows)
- 💡 [Examples](https://amux.dev/docs/examples)

## Development

```bash
just test    # Run tests
just lint    # Lint code
just build   # Build binary
```

See [DEVELOPMENT.md](DEVELOPMENT.md) for contribution guidelines.

## License

MIT - see [LICENSE](LICENSE) for details.
