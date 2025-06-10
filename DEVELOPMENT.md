# Amux Development Guide

## Architecture Overview

Amux follows a clean architecture with clear separation of concerns:

```text
┌─────────────────┬─────────────────┐
│   CLI Layer     │   MCP Server    │  User & AI interfaces
├─────────────────┴─────────────────┤
│          Core Business             │  Workspace & config management
├─────────────────────────────────────┤
│          Infrastructure            │  Git operations, file system
└─────────────────────────────────────┘
```

- **CLI Layer**: Human interface via cobra commands
- **MCP Server**: AI agent interface via Model Context Protocol
- **Core Business**: Shared business logic used by both interfaces
- **Infrastructure**: Low-level operations (git, filesystem)

## Project Structure

```text
amux/
├── cmd/amux/          # Application entry point
│   └── main.go
├── internal/               # Private packages
│   ├── cli/                # Command-line interface
│   │   ├── commands/       # Cobra command definitions
│   │   └── ui/             # Terminal UI utilities
│   ├── core/               # Core business logic
│   │   ├── config/         # Configuration management
│   │   ├── git/            # Git operations wrapper
│   │   └── workspace/      # Workspace lifecycle
│   ├── mcp/                # Model Context Protocol
│   │   ├── server.go       # MCP server implementation
│   │   ├── schema.go       # Tool schema utilities
│   │   └── README.md       # MCP-specific docs
│   └── templates/          # Workspace templates
├── docs/                   # Additional documentation
├── justfile                # Build automation
└── lefthook.yml            # Git hooks configuration
```

## Key Design Decisions

### 1. Interface-First Design

All major components are defined as interfaces:

```go
type WorkspaceManager interface {
    Create(opts CreateOptions) (*Workspace, error)
    Get(id string) (*Workspace, error)
    List(opts ListOptions) ([]*Workspace, error)
    Remove(id string) error
}
```

### 2. Git Worktree Isolation

Each workspace is a separate git worktree:

- Isolated file system
- Independent branch
- No cross-contamination
- Easy cleanup

### 3. MCP Integration

Using mark3labs/mcp-go for Model Context Protocol:

- Type-safe tool definitions
- Struct-to-schema conversion
- Multiple transport support (stdio, HTTP)

### 4. Configuration Layers

```yaml
# .amux/config.yaml
project:
  name: my-project

agents:
  - name: claude
    type: claude-3

mcp:
  transport:
    type: stdio
```

## Development Workflow

### Setup

```bash
# Clone repository
git clone https://github.com/aki/amux
cd amux

# Install dependencies
go mod download

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install mvdan.cc/gofumpt@latest

# Run tests
just test
```

### Building

```bash
# Build binary
just build

# Run development version
just dev ws create test-feature

# Full check (format, lint, test, build)
just all
```

### Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package
go test ./internal/core/workspace
```

### Code Quality

Enforced by Lefthook pre-commit hooks:

- `goimports` - Import formatting
- `gofumpt` - Code formatting
- `golangci-lint` - Linting
- `go vet` - Static analysis
- `go test` - Test execution
- `commitlint` - Commit message format

## Adding New Features

### 1. Adding a New Command

```go
// internal/cli/commands/newcmd.go
var newCmd = &cobra.Command{
    Use:   "new",
    Short: "Do something new",
    RunE:  runNew,
}

func init() {
    rootCmd.AddCommand(newCmd)
}

func runNew(cmd *cobra.Command, args []string) error {
    // Implementation
}
```

### 2. Adding MCP Tools

```go
// Define parameter struct with tags
type MyToolParams struct {
    Name string `json:"name" mcp:"required" description:"Tool name"`
}

// Register in server.go
opts, _ := WithStructOptions("My tool description", MyToolParams{})
s.mcpServer.AddTool(mcp.NewTool("my_tool", opts...), s.handleMyTool)
```

### 3. Extending Workspace Manager

1. Add method to interface
2. Implement in manager
3. Add tests
4. Update CLI if needed

## Common Patterns

### Error Handling

```go
// Wrap errors with context
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Custom errors for specific cases
var ErrWorkspaceNotFound = errors.New("workspace not found")
```

### Path Handling

```go
// Always use filepath for cross-platform paths
path := filepath.Join(baseDir, "subdir", "file.txt")

// Validate paths to prevent traversal
if err := git.ValidateWorktreePath(basePath, userPath); err != nil {
    return fmt.Errorf("invalid path: %w", err)
}
```

### Configuration

```go
// Use functional options pattern
type Option func(*Config)

func WithTimeout(d time.Duration) Option {
    return func(c *Config) {
        c.Timeout = d
    }
}
```

## Debugging

### MCP Server

```bash
# Test with direct JSON-RPC
echo '{"jsonrpc":"2.0","method":"tools/list","id":1}' | amux mcp

# Enable debug logging
export DEBUG=1
amux mcp
```

### Workspace Issues

```bash
# Check git worktree status
git worktree list

# Inspect workspace metadata
cat .amux/workspaces/workspace-*.yaml
```

## Release Process

1. Update version in code
2. Run `just all` to ensure quality
3. Create git tag: `git tag v0.1.0`
4. Push tag: `git push origin v0.1.0`
5. GitHub Actions builds releases

## Contributing

1. Fork the repository
2. Create feature branch
3. Make changes with tests
4. Ensure `just all` passes
5. Submit pull request

### Commit Convention

Using Conventional Commits:

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation
- `refactor:` Code restructuring
- `test:` Test changes
- `chore:` Maintenance

Example: `feat(workspace): add prune command for cleanup`
