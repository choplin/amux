# 🕳️ Amux

[![CI](https://github.com/aki/amux/actions/workflows/ci.yml/badge.svg)](https://github.com/aki/amux/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/aki/amux)](https://goreportcard.com/report/github.com/aki/amux)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **Agent Multiplexer** - Unleash fleets of AI agents in parallel, sandboxed workspaces

Amux provides isolated git worktree-based environments where AI agents can work independently without context mixing. With built-in session management, you can run multiple agents concurrently, attach to their sessions, and manage their lifecycle.

> [!WARNING]
> **🚧 Work in Progress**
>
> This project is actively being developed. Expect frequent updates and potential breaking changes.

## 🚀 Features

- **Concurrent AI Agents**: Run multiple agents in parallel without interference
- **Workspace Isolation**: Each agent works in its own directory and branch
- **Persistent Sessions**: Attach and detach from agent sessions like tmux/screen
- **Bring Your Own Environment**: Works with your existing tools - no containers needed

## 📦 Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/yourusername/amux.git
cd amux

# Build with just (recommended)
just build

# Or with go directly
go build -o bin/amux cmd/amux/main.go

# Or with make (if you don't have just)
go build -o bin/amux cmd/amux/main.go
```

### Binary Releases

Download pre-built binaries from the [releases page](https://github.com/yourusername/amux/releases).

## 🛠️ Usage

### Initialize a Project

```bash
# Initialize Amux in your project
cd your-project
amux init
```

This creates:

- `.amux/config.yaml` - Project configuration
- `.amux/workspaces/` - Workspace metadata directory

### Quick Start

```bash
# Initialize project
amux init

# Create a workspace and run an agent
amux ws create feature-auth --agent claude
amux run claude

# Check running sessions
amux ps

# Attach to a session
amux attach session-abc123
```

### Command Structure

```bash
# Workspace management
amux workspace create <name>    # alias: amux ws create
amux workspace list            # alias: amux ws list
amux workspace get <id>        # alias: amux ws get
amux workspace remove <id>     # alias: amux ws remove
amux workspace prune           # alias: amux ws prune

# Agent management
amux agent run <agent>         # alias: amux run
amux agent list               # alias: amux ps
amux agent attach <session>   # alias: amux attach
amux agent stop <session>
amux agent logs <session>
amux agent config <subcommand>

# MCP server
amux mcp [options]            # Start MCP server
```

### Workspace Management Examples

```bash
# Create a new workspace with a new branch
amux ws create feature-auth --description "Implement authentication"

# Create a workspace using an existing branch
amux ws create bugfix-ui --branch fix/ui-crash --description "Fix UI crash"

# Get details about a specific workspace
amux ws get workspace-abc123

# List all workspaces
amux ws list

# Remove a workspace
amux ws remove workspace-abc123 --force

# Clean up old workspaces
amux ws prune --days 7
```

### Start MCP Server

```bash
# Start with stdio transport (default)
amux mcp

# Start with HTTPS transport
amux mcp --transport https --port 3000 --auth bearer --token secret123
```

## 🤖 MCP Tools for AI Agents

- `workspace_create` - Create isolated workspace (supports existing branches)
- `workspace_list` - List workspaces with optional filtering
- `workspace_get` - Get specific workspace details
- `workspace_remove` - Remove workspace and cleanup
- `workspace_info` - Browse workspace files securely

## 🤖 Agent Multiplexing

Run multiple AI agents concurrently in isolated workspaces:

```bash
# Run agents
amux run claude --workspace feature-auth    # Run Claude in a workspace
amux run gpt --workspace bugfix-api        # Run GPT in another workspace

# Manage sessions
amux ps                                    # List running agents
amux attach session-123                    # Attach to agent session
amux agent stop session-123                # Stop a specific session
amux agent logs session-123                # View session output

# Configure agents
amux agent config add gpt --name "GPT-4" --command "gpt-cli"
amux agent config list                     # List configured agents
```

### Working Context

Each workspace includes context files to help AI agents:

- `background.md` - Task requirements and constraints
- `plan.md` - Implementation approach
- `working-log.md` - Progress tracking
- `results-summary.md` - Final outcomes

Access context path via `$AMUX_CONTEXT_PATH` in agent sessions.

## 📁 Project Structure

```text
amux/
├── cmd/amux/          # CLI entry point
├── internal/
│   ├── adapters/      # External system adapters
│   │   └── tmux/      # Tmux session management
│   ├── cli/           # CLI commands and UI
│   │   └── commands/  # Command implementations
│   ├── core/          # Core business logic
│   │   ├── agent/     # Agent configuration
│   │   ├── config/    # Configuration management
│   │   ├── context/   # Working context management
│   │   ├── git/       # Git operations
│   │   ├── session/   # Session management
│   │   └── workspace/ # Workspace management
│   ├── mcp/           # MCP server implementation
│   └── templates/     # Markdown templates
├── docs/              # Documentation
├── go.mod             # Go module definition
├── go.sum             # Dependency checksums
└── justfile           # Build automation
```

## 🧪 Development

### Prerequisites

- Go 1.22 or later
- tmux (optional, for agent multiplexing)
- [Just](https://github.com/casey/just) (optional, for build automation)

### Building

```bash
# Build binary
just build

# Run tests
just test

# Lint code
just lint

# Format YAML files
just fmt-yaml

# Run all checks (format + lint)
just check
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/core/config
```

## 📚 Documentation

- [Agent Multiplexing Guide](docs/agent-multiplexing.md) - Complete guide to running multiple agents
- [Architecture](docs/architecture.md) - System design and technical details
- [Development Guide](DEVELOPMENT.md) - Setup and contribution guidelines
- [Project Memory](CLAUDE.md) - AI agent context and project knowledge

## License

MIT
