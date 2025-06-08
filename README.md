# 🕳️ AgentCave

> Private development caves for AI agents

AgentCave provides isolated git worktree-based environments where AI agents can work independently without
context mixing. Built in Go for performance and easy deployment.

## 🚀 Features

- **Isolated Workspaces**: Each "cave" is a separate git worktree with its own branch
- **MCP Integration**: Full Model Context Protocol support for AI agent communication
- **Multi-Agent Support**: Multiple agents can work simultaneously in different caves
- **Workspace Management**: Create, list, activate, deactivate, and clean up workspaces
- **Secure File Access**: Path-validated workspace browsing for AI agents
- **Single Binary**: Zero runtime dependencies, easy deployment

## 📦 Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/yourusername/agentcave.git
cd agentcave

# Build with just (recommended)
just build

# Or with go directly
go build -o bin/agentcave cmd/agentcave/main.go

# Or with make (if you don't have just)
go build -o bin/agentcave cmd/agentcave/main.go
```

### Binary Releases

Download pre-built binaries from the [releases page](https://github.com/yourusername/agentcave/releases).

## 🛠️ Usage

### Initialize a Project

```bash
# Initialize AgentCave in your project
cd your-project
agentcave init
```

This creates:

- `.agentcave/config.yaml` - Project configuration
- `.agentcave/workspaces/` - Workspace metadata directory

### Workspace Management

```bash
# Create a new workspace with a new branch
agentcave workspace create feature-auth --description "Implement authentication"

# Create a workspace using an existing branch
agentcave workspace create bugfix-ui --branch fix/ui-crash --description "Fix UI crash"

# List all workspaces
agentcave workspace list

# Activate a workspace
agentcave workspace activate workspace-abc123

# Remove a workspace
agentcave workspace remove workspace-abc123 --force

# Clean up old workspaces
agentcave workspace cleanup --days 7
```

### Start MCP Server

```bash
# Start with stdio transport (default)
agentcave serve

# Start with HTTPS transport
agentcave serve --transport https --port 3000 --auth bearer --token secret123
```

## 🤖 MCP Tools for AI Agents

- `workspace_create` - Create isolated workspace (supports existing branches)
- `workspace_list` - List workspaces with optional filtering  
- `workspace_get` - Get specific workspace details
- `workspace_activate`/`workspace_deactivate` - Manage workspace states
- `workspace_remove` - Remove workspace and cleanup
- `workspace_info` - Browse workspace files securely

## 📁 Project Structure

```text
agentcave/
├── cmd/agentcave/      # CLI entry point
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

# Run all checks
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

## License

MIT
