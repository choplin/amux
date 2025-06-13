# 🕳️ Amux

[![CI](https://github.com/choplin/amux/actions/workflows/ci.yml/badge.svg)](https://github.com/choplin/amux/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/choplin/amux)](https://goreportcard.com/report/github.com/choplin/amux)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **Agent Multiplexer** - Unleash fleets of AI agents in parallel, sandboxed workspaces

Amux provides isolated git worktree-based environments where AI agents can work independently without context mixing.
With built-in session management, you can run multiple agents concurrently, attach to their sessions, and manage
their lifecycle.

## 📦 v0.1.0 Release Status

- ✅ **Workspace Management**: Fully functional and ready for use
- 🚧 **Session/Agent Features**: Preview release - foundational structure in place, full functionality coming soon

For v0.1.0, we recommend starting with the workspace features which provide stable, isolated development environments.

> [!WARNING]
> **🚧 Alpha Release**
>
> This software is in alpha stage. Features may be incomplete, unstable, or change significantly.
> Expect bugs and breaking changes until the 1.0 release.

## 🚀 Features

- **Concurrent AI Agents**: Run multiple agents in parallel without interference
- **Workspace Isolation**: Each agent works in its own directory and branch
- **Persistent Sessions**: Attach and detach from agent sessions like tmux/screen
- **Bring Your Own Environment**: Works with your existing tools - no containers needed

## 📦 Installation

### Homebrew (macOS and Linux)

```bash
brew tap choplin/amux
brew install amux
```

### From Source

```bash
# Clone the repository
git clone https://github.com/choplin/amux.git
cd amux

# Build with just (recommended)
just build

# Or with go directly
go build -o bin/amux cmd/amux/main.go

# Or with make (if you don't have just)
go build -o bin/amux cmd/amux/main.go
```

### Binary Releases

Download pre-built binaries from the [releases page](https://github.com/choplin/amux/releases).

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

# Run an agent (auto-creates workspace if needed)
amux run claude

# Or create a specific workspace first
amux ws create feature-auth
amux run claude --workspace feature-auth

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
amux workspace show <id>       # alias: amux ws show
amux workspace remove <id>     # alias: amux ws remove
amux workspace prune           # alias: amux ws prune

# Session management
amux session run <agent>       # alias: amux run
amux session list             # alias: amux ps
amux session attach <session> # alias: amux attach
amux session stop <session>
amux session remove <session> # alias: amux session rm
amux session logs <session>   # View session output
amux session logs -f <session> # Follow logs (tail -f behavior)
amux tail <session>           # alias: amux session logs -f

# Agent configuration
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

# Show details about a specific workspace
amux ws show workspace-abc123

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

### Using MCP Features

#### Accessing Resources

```bash
# In your AI agent, you can read resources like:
# Read workspace list
GET amux://workspace

# Read specific workspace details
GET amux://workspace/ws-feature-auth-123

# Browse files in a workspace
GET amux://workspace/ws-feature-auth-123/files
GET amux://workspace/ws-feature-auth-123/files/src/auth.go

# Read workspace context
GET amux://workspace/ws-feature-auth-123/context
```

#### Using Prompts

```bash
# Start working on an issue
PROMPT start-issue-work {
  "issue_number": "42",
  "issue_title": "Add authentication system"
}

# Prepare a PR when done
PROMPT prepare-pr {
  "pr_title": "feat: implement JWT authentication"
}

# Review workspace state
PROMPT review-workspace {
  "workspace_id": "ws-feature-auth-123"
}
```

## 🤖 MCP Integration for AI Agents

### MCP Resources (Read-only Data)

Amux provides structured read-only data through MCP Resources:

#### Static Resources

- `amux://workspace` - List all workspaces with metadata and resource URIs

#### Dynamic Resources (Per Workspace)

- `amux://workspace/{id}` - Detailed workspace information including paths
- `amux://workspace/{id}/files` - Browse workspace directory structure
- `amux://workspace/{id}/files/{path}` - Read specific files
- `amux://workspace/{id}/context` - Access workspace context.md file

Example resource URIs:

```text
amux://workspace/ws-abc123
amux://workspace/ws-abc123/files
amux://workspace/ws-abc123/files/src/main.go
amux://workspace/ws-abc123/context
```

### MCP Tools (Actions)

- `workspace_create` - Create isolated workspace (supports existing branches)
- `workspace_remove` - Remove workspace and cleanup

### MCP Prompts (Guided Workflows)

Amux provides prompts to guide AI agents through common workflows:

- **`start-issue-work`** - Start working on an issue with structured approach
  - Parameters: `issue_number` (required), `issue_title`, `issue_url`
  - Guides through requirements clarification and planning

- **`prepare-pr`** - Prepare code for pull request submission
  - Parameters: `pr_title`, `pr_description` (optional)
  - Ensures tests pass and code is properly formatted

- **`review-workspace`** - Analyze workspace state and suggest next steps
  - Parameters: `workspace_id` (required)
  - Shows workspace age, branch status, and recommended actions

## 🤖 Agent Multiplexing

Run multiple AI agents concurrently in isolated workspaces:

```bash
# Run agents
amux run claude --workspace feature-auth    # Run Claude in a workspace
amux run gpt --workspace bugfix-api        # Run GPT in another workspace

# Manage sessions
amux ps                                    # List running agents
amux attach session-123                    # Attach to agent session
amux session stop session-123              # Stop a specific session
amux session remove session-123            # Remove a stopped session
amux session logs session-123              # View session output
amux session logs -f session-123           # Follow logs in real-time
amux tail session-123                      # Shortcut for follow logs

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

### Agent Communication

Amux provides a mailbox system for asynchronous communication with running agents:

```bash
# Send messages to an agent
amux mailbox send s1 "Please focus on the authentication module"
amux mb send s1 "Fix the test failures"   # Short alias
amux mailbox send s1 --file plan.md       # From file
echo "urgent" | amux mb send s1           # From stdin

# Receive latest message from agent
amux mailbox recv s1                      # Show latest with metadata
amux mb recv s1 -q                        # Just the content

# List message files with indices
amux mailbox list s1                      # Shows numbered list
amux mb ls s1                             # Short alias

# Show specific messages
amux mailbox show s1                      # Show all with previews
amux mb show s1 3                         # Show message #3
amux mb show s1 latest                    # Latest from agent
amux mb show s1 latest --in               # Latest to agent
amux mb show s1 --tail 5                  # Last 5 messages
```

Each session has a mailbox directory at `.amux/mailbox/{session-id}/` with:

- `in/` - Messages TO the agent
- `out/` - Messages FROM the agent
- `context.md` - Mailbox instructions

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
│   │   ├── mailbox/   # Agent communication
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

- [Documentation Guide](docs/README.md) - Overview of our documentation structure
- [MCP Integration](docs/mcp.md) - Model Context Protocol resources, tools, and prompts
- [Agent Multiplexing Guide](docs/agent-multiplexing.md) - Complete guide to running multiple agents
- [Architecture](docs/architecture.md) - System design and technical details
- [Architecture Decision Records](docs/adr/) - Design decisions and rationale
- [Development Guide](DEVELOPMENT.md) - Setup and contribution guidelines
- [Project Memory](CLAUDE.md) - AI agent context and project knowledge

## License

MIT
