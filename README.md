# 🕳️ Amux
>
> Private development caves for AI agents
Amux provides isolated git worktree-based environments where AI agents can work independently without
context mixing. Built in Go for performance and easy deployment.

## 🚀 Features

- **Isolated Workspaces**: Each "cave" is a separate git worktree with its own branch
- **MCP Integration**: Full Model Context Protocol support for AI agent communication
- **Multi-Agent Support**: Multiple agents can work simultaneously in different caves
- **Workspace Management**: Create, list, and clean up workspaces with ease
- **Secure File Access**: Path-validated workspace browsing for AI agents
- **Single Binary**: Zero runtime dependencies, easy deployment

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

### Command Structure

```bash
# Workspace management
amux workspace create <name>    # alias: amux ws create
amux workspace list            # alias: amux ws list
amux workspace get <id>        # alias: amux ws get
amux workspace remove <id>     # alias: amux ws remove
amux workspace prune           # alias: amux ws prune

# Agent management (future)
amux agent run <agent>         # alias: amux run
amux agent list               # alias: amux ps
amux agent attach <session>   # alias: amux attach
amux agent stop <session>     # no alias

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

## 🎯 Future: Agent Multiplexing

Amux is designed to support running multiple AI agents concurrently:

```bash
# Future functionality
amux run claude --workspace feature-auth    # Run Claude in a workspace
amux ps                                    # List running agents
amux attach claude-session-123             # Attach to agent session
```

## 📁 Project Structure

```text
amux/
├── cmd/amux/      # CLI entry point
├── internal/
│   ├── cli/           # CLI commands and UI
│   ├── core/          # Core business logic
│   │   ├── config/    # Configuration management
│   │   ├── git/       # Git operations
│   │   └── workspace/ # Workspace management
│   ├── mcp/           # MCP server implementation
│   └── templates/     # Markdown templates
├── go.mod             # Go module definition
├── go.sum             # Dependency checksums
└── justfile           # Build automation
```

## 🧪 Development

### Prerequisites

- Go 1.22 or later
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

- [Development Guide](DEVELOPMENT.md) - Architecture, setup, and contribution guidelines
- [Project Memory](CLAUDE.md) - AI agent context and project knowledge

## License

MIT
