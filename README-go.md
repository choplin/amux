# AgentCave Go Implementation

This is the Go implementation of AgentCave, providing private development caves for AI agents through isolated git worktree-based environments.

## Features

- **Workspace Isolation**: Separate git worktrees prevent context mixing between agents
- **MCP Integration**: Model Context Protocol server for AI agent communication
- **Multi-Agent Support**: Multiple AI agents working simultaneously in different workspaces
- **Secure File Access**: Path-validated workspace browsing for AI agents
- **Multiple Transports**: Support for stdio and HTTP/HTTPS transports

## Installation

### Prerequisites

- Go 1.21 or higher
- Git
- A git repository for your project

### Building from Source

```bash
# Clone the repository
git clone https://github.com/aki/agentcave.git
cd agentcave

# Initialize and build
make init
make build

# Or using just
just init
just build
```

The binary will be created in the `bin/` directory.

### Installing

```bash
# Install to GOPATH/bin
make install

# Or using just
just install
```

## Usage

### Initialize AgentCave in Your Project

```bash
cd your-project
agentcave init
```

This creates:
- `.agentcave/config.yaml` - Project configuration
- `.agentcave/workspaces/` - Workspace metadata directory

### Start the MCP Server

```bash
# Start with stdio transport (default)
agentcave serve

# Start with HTTP transport
agentcave serve --transport http --port 3000

# Start with authentication
agentcave serve --transport http --port 3000 --auth bearer --auth-token YOUR_TOKEN
```

### Manage Workspaces

```bash
# Create a new workspace
agentcave workspace create feature-auth --agent claude --description "Implement authentication"

# List all workspaces
agentcave workspace list

# List active workspaces only
agentcave workspace list --status active

# Activate/deactivate a workspace
agentcave workspace activate workspace-abc123
agentcave workspace deactivate workspace-abc123

# Remove a workspace
agentcave workspace remove workspace-abc123 --force

# Clean up old idle workspaces
agentcave workspace cleanup --days 7
agentcave workspace cleanup --days 7 --dry-run  # Preview what would be removed
```

## MCP Tools for AI Agents

When connected via MCP, AI agents have access to these tools:

### `cave_create`
Create a new isolated workspace.

```json
{
  "name": "feature-name",
  "baseBranch": "main",
  "agentId": "claude",
  "description": "Implement feature X"
}
```

### `cave_list`
List all workspaces with optional status filtering.

```json
{
  "status": "active"  // optional: "active" or "idle"
}
```

### `cave_get`
Get details about a specific workspace.

```json
{
  "cave_id": "workspace-abc123"
}
```

### `cave_activate` / `cave_deactivate`
Change workspace status.

```json
{
  "cave_id": "workspace-abc123"
}
```

### `cave_remove`
Remove a workspace and its resources.

```json
{
  "cave_id": "workspace-abc123"
}
```

### `workspace_info`
Browse and read files in a workspace.

```json
{
  "cave_id": "workspace-abc123",
  "path": "src/main.go"  // optional, defaults to root
}
```

## Project Structure

```
agentcave/
├── cmd/agentcave/          # Main entry point
├── internal/
│   ├── cli/                # CLI implementation
│   │   ├── commands/       # Command implementations
│   │   └── ui/             # Terminal UI helpers
│   ├── core/               # Core business logic
│   │   ├── config/         # Configuration management
│   │   ├── git/            # Git operations
│   │   └── workspace/      # Workspace management
│   ├── mcp/                # MCP server implementation
│   └── templates/          # Workspace template files
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── Makefile                # Build automation
└── justfile                # Alternative build automation
```

## Development

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Format and lint
make check
```

### Development Commands

```bash
# Run without building
make dev init
make dev serve
make dev workspace create test-workspace

# Using just
just dev init
just dev serve
just dev workspace create test-workspace
```

## Configuration

The configuration file is located at `.agentcave/config.yaml`:

```yaml
version: "1.0"
project:
  name: "your-project"
  repository: "https://github.com/user/repo.git"
  defaultAgent: "claude"
mcp:
  transport:
    type: "stdio"  # or "http"
    http:
      port: 3000
      auth:
        type: "bearer"  # or "basic" or "none"
        bearer: "your-token"
agents:
  claude:
    name: "Claude"
    type: "claude"
```

## Architecture

### Git Worktree Integration

AgentCave uses git worktrees to provide isolated environments:
- Each workspace is a separate worktree in `.worktrees/workspace-{id}/`
- Each workspace has its own branch: `agentcave/workspace-{id}`
- Workspaces are completely isolated from each other

### MCP Server

The MCP server provides:
- JSON-RPC 2.0 protocol implementation
- Multiple transport support (stdio, HTTP/HTTPS)
- Authentication (bearer token, basic auth)
- CORS support for web-based clients

### Workspace Lifecycle

1. **Creation**: Creates worktree, branch, and initial context files
2. **Active**: Workspace is being actively used by an agent
3. **Idle**: Workspace is inactive but preserved
4. **Removal**: Worktree and branch are deleted

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.