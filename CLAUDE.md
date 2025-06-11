# Amux Project Memory

## Project Overview

**Amux** provides private development caves for AI agents. It's a workspace management tool that creates isolated
git worktree-based environments where AI agents can work independently without context mixing.

## Cave Concept

A **Cave** is an isolated development environment where AI agents work autonomously. While physically implemented as
git worktree workspaces, a workspace becomes a "Cave" when it contains the Working Context files that enable AI agents
to work effectively:

### Working Context

The Working Context consists of four markdown files that guide AI agent work:

1. **background.md** - Project requirements and constraints
   - Written at task start
   - Contains requirements, issues, constraints, dependencies
   - Source information from tickets, user interviews

2. **plan.md** - Implementation approach and task breakdown
   - Written before coding
   - Technical decisions, risk assessment
   - Concrete task breakdown

3. **working-log.md** - Real-time progress and decision records
   - Updated continuously during work
   - Timestamped progress entries
   - Key decisions and rationale
   - Challenges and resolutions

4. **results-summary.md** - Final outcomes for review/PR
   - Written at completion
   - Summary of implementation
   - Key changes and impact
   - Suitable for PR descriptions

These files ensure AI agents maintain context, make informed decisions, and produce reviewable work.

## Project Status

### Completed Features

- ✅ **Renamed to Amux** (ADR-004) - Changed from AgentCave to Amux (Agent Multiplexer)
- ✅ **Command Structure** (ADR-005) - Implemented subcommand structure with aliases
  - Workspace commands: `amux ws create/list/get/remove/prune`
  - Agent commands: `amux agent run/list/attach/stop` (structure only)
  - Global aliases: `amux run/ps/attach`
- ✅ **Core Functionality** - Workspace management, MCP server, git integration
- ✅ **Single Binary** - Zero runtime dependencies

### Recently Completed

- ✅ **Agent Multiplexing** (ADR-003) - Running multiple AI agents concurrently
  - Session management with tmux backend
  - Agent configuration system
  - CLI commands for agent lifecycle
  - Working context management
  - Full test coverage with mock adapter
- ✅ **Session Mailbox System** (Issue #4) - Agent communication
  - File-based mailbox for asynchronous messaging
  - `amux tell` command for sending messages
  - `amux peek` command for viewing mailbox
  - Automatic mailbox initialization on session creation
  - Full test coverage

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
amux/
├── cmd/amux/        # Entry point
├── internal/
│   ├── cli/              # CLI commands and UI
│   ├── core/             # Core business logic
│   │   ├── config/       # Configuration management
│   │   ├── git/          # Git operations
│   │   ├── mailbox/      # Agent communication
│   │   └── workspace/    # Workspace management
│   ├── mcp/              # MCP server implementation
│   └── templates/        # Workspace templates
└── docs/                 # Documentation
```

### Key Commands

- `amux init` - Initialize project
- `amux ws create <name>` - Create workspace
- `amux ws list` - List workspaces (alias: `ls`)
- `amux ws get <id>` - Get workspace details
- `amux ws remove <id>` - Remove workspace (alias: `rm`)
- `amux ws prune` - Clean old workspaces
- `amux mcp` - Start MCP server
- `amux agent <cmd>` - Agent management
- `amux run/ps/attach` - Agent shortcuts
- `amux mailbox <cmd>` - Mailbox commands (alias: `mb`)
  - `amux mailbox tell <session> <msg>` - Send message to agent
  - `amux mailbox peek <session>` - View agent mailbox

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
- [docs/adr/](docs/adr/) - Architecture Decision Records (immutable)
- [.claude/planning/](.claude/planning/) - Implementation plans for AI agents
- [.claude/archive/](.claude/archive/) - Historical context and migration notes

## Working Context Management

For each work session, maintain context in `.claude/context/{work_name}/`:

- `background.md` - Requirements for current work
- `plan.md` - Implementation approach
- `working-log.md` - Progress and decisions
- `results-summary.md` - Summary of changes

## Important Notes

1. **Workspace Isolation**: Each workspace is a separate git worktree
2. **No Manual Status Tracking**: Use filesystem timestamps instead
3. **Name Resolution**: All commands accept both workspace names and IDs
4. **Path Security**: Workspace file access is path-validated
5. **Single Binary**: No runtime dependencies for easy deployment
6. **Git Commits**: NEVER commit without explicit user confirmation
7. **ADRs are Immutable**: Never modify existing ADRs. To change a decision, create a new ADR that mentions
   "This supersedes ADR-XXX"
8. **Implementation Plans**: Detailed plans go in `.claude/planning/`, not in ADRs
9. **Working Context**: Templates exist in `internal/templates/` but not yet integrated

## Documentation Structure

- **Architecture Decision Records (ADRs)**: Located in `docs/adr/` for significant design decisions
- **Archive Memories**: Located in `.claude/archive/` (gitignored) for historical context and past work sessions
