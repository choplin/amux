# AgentCave Project Memory

## Project Overview

**AgentCave** provides private development caves for AI agents. It's a workspace management tool that creates isolated
git worktree-based environments where AI agents can work independently without context mixing.

## Current Implementation (Go)

### Technology Stack

- **Language**: Go
- **MCP Integration**: Official Go MCP SDK (mark3labs/mcp-go)
- **Code Quality**: golangci-lint, yamlfmt
- **Git Hooks**: Lefthook with commitlint
- **Build Tool**: Just (justfile)
- **Git Management**: go-git library

### Architecture

```text
agentcave/
├── cmd/agentcave/        # Entry point
├── internal/
│   ├── cli/              # CLI commands and UI
│   ├── core/             # Core business logic
│   │   ├── config/       # Configuration management
│   │   ├── git/          # Git operations
│   │   └── workspace/    # Workspace management
│   ├── mcp/              # MCP server implementation
│   └── templates/        # Workspace templates
└── docs/                 # Documentation
```

### Key Commands

- `agentcave init` - Initialize project
- `agentcave ws create <name>` - Create workspace
- `agentcave ws ls` - List workspaces
- `agentcave ws rm <name/id>` - Remove workspace
- `agentcave ws prune` - Clean old workspaces
- `agentcave mcp` - Start MCP server

### MCP Tools

1. `workspace_create` - Create isolated workspace
2. `workspace_list` - List all workspaces
3. `workspace_get` - Get workspace details
4. `workspace_remove` - Remove workspace
5. `workspace_info` - Browse workspace files

## Development Patterns

### Code Style

- Use Go interfaces for abstraction
- Prefer composition over inheritance
- Keep packages focused and cohesive
- Use dependency injection

### Error Handling

- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- Return early on errors
- Use custom error types sparingly

### Testing

- Table-driven tests preferred
- Mock interfaces, not implementations
- Test files alongside implementation

## Key Documents

- [README.md](README.md) - User guide
- [DEVELOPMENT.md](DEVELOPMENT.md) - Developer documentation

## Working Context

For each work session, maintain context in `.claude/context/{work_name}/`:

- `background.md` - Requirements for current work
- `plan.md` - Implementation approach
- `working-log.md` - Progress and decisions
- `results-summary.md` - Summary of changes

## Migration History

### From TypeScript to Go (December 2024)

- **Reason**: Go lacks official MCP SDK, TypeScript has official SDK
- **Benefits**: Better performance, single binary, cross-platform
- **Maintained**: All features, test coverage, code quality

### Terminology Evolution

- Originally "AiSquad" → renamed to "AgentCave"
- "Cave" = Isolated Workspace with Working Context
- Focus shifted from task management to workspace management

## Important Notes

1. **Workspace Isolation**: Each workspace is a separate git worktree
2. **No Manual Status Tracking**: Use filesystem timestamps instead
3. **Name Resolution**: All commands accept both workspace names and IDs
4. **Path Security**: Workspace file access is path-validated
5. **Single Binary**: No runtime dependencies for easy deployment
6. **Git Commits**: NEVER commit without explicit user confirmation
