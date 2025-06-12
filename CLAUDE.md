# Amux Project Memory

## Project Overview

**Amux** (Agent Multiplexer) is a workspace management tool that creates isolated git worktree-based environments
where AI agents can work independently. It's now functionally complete and ready for dogfooding!

## 🚀 Dogfooding Workflow

### Initial Setup

1. **Initialize Project** (if not done)

   ```bash
   amux init
   ```

2. **Configure Claude Code MCP**
   Add to your MCP settings:

   ```json
   {
     "mcpServers": {
       "amux": {
         "command": "/Users/aki/workspace/amux/bin/amux",
         "args": ["mcp", "--root-dir", "/Users/aki/workspace/amux"],
         "env": {}
       }
     }
   }
   ```

### Typical AI Agent Workflow

When working on GitHub issues with AI agents (like Claude), follow this workflow:

1. **User assigns issue** → "Work on issue #30" or "Pick an open issue to work on"

2. **AI creates workspace** via MCP:

   ```typescript
   workspace_create({
     name: "fix-issue-30",
     baseBranch: "main", // Always specify base branch
     description: "Standardize console output (#30)",
   });
   ```

3. **AI works in the workspace**:

   - Uses `workspace_info` to browse files
   - Makes changes within the workspace
   - Tests changes in isolation
   - Creates CLAUDE.local.md for task context if needed

4. **AI creates pull request**:

   ```bash
   # In the workspace directory
   git add .
   git commit -m "fix: standardize console output"
   gh pr create --draft --base main --title "fix: standardize console output (#30)"
   ```

5. **After PR is merged**, AI cleans up:

   ```typescript
   workspace_remove({ workspace_id: "fix-issue-30" });
   ```

### Available MCP Tools for AI Agents

When working in Claude Code, you can use these amux tools:

1. **workspace_create** - Create isolated workspace

   ```typescript
   workspace_create({
     name: "feature-auth",
     description: "Implement authentication",
     branch?: "existing-branch",  // optional
     agentId?: "claude"          // optional
   })
   ```

2. **workspace_list** - List all workspaces

   ```typescript
   workspace_list(); // Shows ID, name, branch, created time
   ```

3. **workspace_get** - Get workspace details

   ```typescript
   workspace_get({ workspace_id: "1" }); // Use name or ID
   ```

4. **workspace_info** - Browse workspace files

   ```typescript
   workspace_info({
     workspace_id: "1",
     path?: "src/"  // optional path within workspace
   })
   ```

5. **workspace_remove** - Clean up workspace

   ```typescript
   workspace_remove({ workspace_id: "1" });
   ```

## Current Functional Features

### ✅ Core Functionality

- **Workspace Management**: Create, list, get, remove workspaces
- **Git Integration**: Each workspace is a separate git worktree
- **MCP Server**: Full integration with Claude Code
- **Short IDs**: Simple numeric IDs (1, 2, 3) instead of UUIDs
- **Session Mailbox**: Communication system for agents (CLI only for now)

### ⚠️ Limitations

- **Agent Commands**: Structure exists but not fully implemented
  - `amux run/ps/attach` - Planned but not functional yet
  - Use MCP tools through Claude Code instead
- **Working Context**: Templates exist but not auto-created yet
  - Manually create context files if needed
- **Log Tailing**: Not implemented yet (issue #6)

## Development Workflow

### Workspace-Specific Context

Each workspace can have its own `CLAUDE.local.md` file for workspace-specific instructions and context:

```markdown
# CLAUDE.local.md (in workspace root)

- Task-specific requirements
- Design decisions for this feature
- TODO items for this workspace
- Any workspace-specific instructions
```

This file is automatically loaded when AI agents work in the workspace, supplementing the global CLAUDE.md.

### For Bug Fixes

```bash
# 1. Create workspace
amux ws create fix-issue-30 --description "Standardize console output"

# 2. (Optional) Add workspace-specific context
cd .amux/workspaces/workspace-fix-issue-30-*
echo "# Fix Issue #30\n\nReplace all fmt.Print* with ui.Output methods" > CLAUDE.local.md

# 3. Work in Claude Code using MCP tools
# 4. Test your changes
just test

# 5. Create PR when ready
git add .
git commit -m "fix: standardize console output"
gh pr create --draft
```

### For New Features

```bash
# 1. Create feature workspace
amux ws create feat-log-tail --description "Add log tailing command"

# 2. (Optional) Create detailed workspace context
cd .amux/workspaces/workspace-feat-log-tail-*
cat > CLAUDE.local.md << 'EOF'
# Log Tailing Feature

## Requirements
- Real-time log output from agent sessions
- Support for `amux tail <session-id>`
- Handle tmux output streaming

## Design Notes
- Use tmux capture-pane for reading output
- Stream updates every 100ms
- Support --follow flag for continuous tailing
EOF

# 3. Use Claude Code with amux MCP tools
# 4. Run tests and checks
just check
```

## Project Structure

```text
amux/
├── .amux/                         # Amux data directory
│   ├── config.yaml               # Project configuration
│   ├── workspaces/               # Workspace directories (git worktrees)
│   │   └── workspace-{name}-*/   # Isolated workspace directories
│   │       └── CLAUDE.local.md   # Workspace-specific context (optional)
│   └── mailbox/                  # Agent mailboxes (when using CLI)
├── cmd/amux/                     # CLI entry point
├── internal/                     # Core implementation
├── docs/                         # Documentation
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

## Important Notes

1. **MCP Connection**: Always use `--root-dir` flag when starting MCP server
2. **Workspace Names**: Use descriptive names with prefixes (fix-, feat-, chore-)
3. **IDs**: Both names and numeric IDs work for all commands
4. **Git Integration**: Each workspace is a real git worktree - use git normally
5. **No Auto-commit**: Amux never commits without explicit user action
6. **Context Loading**: AI agents automatically load both CLAUDE.md (global) and CLAUDE.local.md
   (workspace-specific) if present

## Workspace Management Safety

When working in amux workspaces:

- NEVER remove a workspace while your current directory is inside it
- Always `cd` out of the workspace directory before running `workspace_remove`
- If you need to clean up a workspace after completing work:
  1. First change directory to the main repository: `cd /Users/aki/workspace/amux`
  2. Then remove the workspace: `workspace_remove({ workspace_id: "..." })`

## Quick Reference

### Essential Commands

```bash
# Initialize project
amux init

# Workspace management
amux ws create <name> [--description "..."] [--branch existing-branch]
amux ws list
amux ws get <id-or-name>
amux ws remove <id-or-name>

# Start MCP server
amux mcp --root-dir /path/to/project

# Check version
amux version
```

### Coming Soon

- Real-time session management (`amux run/ps/attach`)
- Automatic working context creation
- Log tailing (`amux tail`)
- Session status tracking

## Troubleshooting

### MCP Connection Issues

If Claude Code can't connect:

1. Ensure `--root-dir` points to valid git repository
2. Check amux binary path is absolute in MCP config
3. Restart Claude Code after config changes

### Workspace Issues

- Can't create workspace? Check you're in initialized project (`amux init`)
- Workspace not found? Use `amux ws list` to see all workspaces
- Need to clean up? Use `amux ws prune --days 7`

---

**Ready to dogfood!** Create workspaces for all your amux development tasks and help us improve the tool by using it.
