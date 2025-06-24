# Amux Project Memory

## Project Overview

**Amux** (Agent Multiplexer) is a workspace management tool that creates isolated git worktree-based environments
where AI agents can work independently. It's now functionally complete and ready for dogfooding!

## Current Functional Features

### ✅ Core Functionality

- **Workspace Management**: Create, list, get, remove workspaces
- **Git Integration**: Each workspace is a separate git worktree
- **MCP Server**: Full integration with Claude Code
- **Short IDs**: Simple numeric IDs (1, 2, 3) instead of UUIDs
- **Context Files**: Workspace context files at `.amux/workspaces/{id}/context.md`

### ✅ New Features

- **Session Status Tracking**: Real-time activity monitoring for agent sessions
  - Shows "busy", "idle", or "stuck" status in `amux ps` and `amux status`
  - Helps identify when agents need assistance

### ⚠️ Limitations

- **Agent Commands**: Structure exists but not fully implemented
  - `amux run/ps/attach` - Planned but not functional yet
  - Use MCP tools through Claude Code instead
- **Working Context**: Templates exist but not auto-created yet
  - Manually create context files if needed
- **Log Tailing**: Not implemented yet (issue #6)

## Project Structure

```text
amux/
├── .amux/                         # Amux data directory
│   ├── config.yaml               # Project configuration
│   ├── workspaces/               # Workspace storage
│   │   └── workspace-{id}/       # Workspace directories
│   │       ├── workspace.yaml    # Workspace metadata
│   │       ├── storage/          # Workspace storage (optional)
│   │       └── worktree/         # Git worktree (clean workspace)
│   └── sessions/                 # Session data and storage
├── cmd/amux/                     # CLI entry point
├── internal/                     # Core implementation
├── docs/                         # Documentation (stable docs only - no planning/temporary files)
└── CLAUDE.md                     # Global project context
```

## Development Standards

### Code Style Rules

1. **Go Code Formatting**: Use `goimports` and `gofumpt` via golangci-lint for consistent formatting
2. **Error Messages**: Start with lowercase, no punctuation at end
3. **Commit Messages**: Follow conventional commits (fix:, feat:, chore:, etc.)
4. **Line Length**: Keep lines under 120 characters when reasonable
5. **Comments**: Exported functions must have comments starting with function name

### Standard Development Tasks

```bash
# Essential commands for development
just build          # Build the binary
just test           # Run all tests
just test-coverage  # Run tests with coverage report
just lint           # Run golangci-lint
just fmt            # Format Go code with goimports and gofumpt
just fmt-yaml       # Format YAML files
just check          # Run all checks (fmt + lint)

# Full development cycle
just check          # Run before committing
just test           # Ensure tests pass
git commit          # Triggers pre-commit hooks automatically
```

### Pre-commit Hooks

The project uses Lefthook for git hooks:

- **commitlint**: Enforces conventional commit messages
- **markdown-lint**: Checks markdown files
- **golangci-fmt**: Formats Go code with goimports and gofumpt
- **yamlfmt**: Formats YAML files
- Tests and linting run automatically

### Code Guidelines

#### When Contributing

1. **Use Workspaces**: Always create a workspace for your changes
2. **Follow Conventions**: Match existing code style and patterns
3. **Test Changes**: Run `just test` before committing
4. **Check Quality**: Run `just check` for linting and formatting
5. **Update Tests**: Add/update tests for new functionality

#### Error Handling

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create workspace: %w", err)
}

// Use lowercase messages
return fmt.Errorf("workspace not found")  // Good
return fmt.Errorf("Workspace not found.")  // Bad
```

#### Testing

- Write table-driven tests
- Mock interfaces, not implementations
- Keep test files next to implementation
- Aim for >80% coverage on new code

#### Pull Request Guidelines

- **Always create draft PRs** using `gh pr create --draft`
- **Never push directly to main branch**
- **Create feature branches** in workspaces using amux
- **Include issue references** in PR descriptions (NOT in titles)
- **Run all checks** before creating PR (`just check && just test`)

## Important Notes

1. **MCP Connection**: Always use `--git-root` flag when starting MCP server
2. **Workspace Names**: Use descriptive names with prefixes (fix-, feat-, chore-)
3. **IDs**: Both names and numeric IDs work for all commands
4. **Git Integration**: Each workspace is a real git worktree - use git normally
5. **No Auto-commit**: Amux never commits without explicit user action
6. **Context Files**: Workspace context files are stored at `.amux/workspaces/{id}/context.md`
   - Path is available via `amux ws show` or MCP workspace resources
   - Files are optional - create only when documentation is needed
   - Located outside git worktree to keep repository clean

## Quick Reference

### Essential Commands

```bash
# Initialize project
amux init

# Workspace management
amux ws create <name> [--description "..."] [--branch existing-branch]
amux ws list
amux ws show <id-or-name>
amux ws remove <id-or-name>

# Start MCP server
amux mcp --git-root /path/to/project

# Check version
amux version
```

### Coming Soon

- Real-time session management (`amux run/ps/attach`)
- Automatic working context creation
- Log tailing (`amux tail`)
- Session status tracking

## Important: Directory-Specific Instructions

**Always check for CLAUDE.md files in subdirectories** before working in them. These contain specific instructions that override general guidelines.

For example:

- `docs/adr/CLAUDE.md` - Instructions for writing ADRs
- Other directories may have their own CLAUDE.md files

**Always read and follow these directory-specific instructions carefully!**

### About the `docs/` Directory

The `docs/` directory should only contain **stable documentation** that reflects the current state of the codebase:

- Architecture Decision Records (ADRs) in `docs/adr/`
- User guides and references
- API documentation
- Any documentation that describes how the project currently works

**DO NOT** put the following in `docs/`:

- Planning documents
- Temporary design documents
- Work-in-progress documentation
- Meeting notes or brainstorming documents

For temporary planning documents, use:

- Workspace storage (via `workspace_storage_write` MCP tool)
- GitHub issues or PR descriptions
- External planning tools

## Troubleshooting

### MCP Connection Issues

If Claude Code can't connect:

1. Ensure `--git-root` points to valid git repository
2. Check amux binary path is absolute in MCP config
3. Restart Claude Code after config changes

### Workspace Issues

- Can't create workspace? Check you're in initialized project (`amux init`)
- Workspace not found? Use `amux ws list` to see all workspaces
- Need to clean up? Use `amux ws prune --days 7`

---

@CLAUDE.local.md
