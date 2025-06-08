# AI Agent Instructions

This workspace ({{.WorkspaceID}}) is your isolated development environment. You are working on branch `{{.Branch}}`.

## Workspace Guidelines

1. **Context Files**: Update the context files in `.agentcave/context/` as you work:
   - `background.md` - Document the task requirements at the start
   - `plan.md` - Outline your implementation approach before coding
   - `working-log.md` - Log progress and decisions as you work
   - `results-summary.md` - Summarize outcomes when complete

2. **Git Workflow**:
   - This is an isolated git worktree on branch `{{.Branch}}`
   - Make atomic commits with clear messages
   - Keep the workspace clean and organized

3. **Development Practice**:
   - Follow the project's coding standards
   - Write tests for new functionality
   - Update documentation as needed
   - Keep dependencies minimal

## Available Tools

You have access to the following MCP tools:

- `workspace_info` - Browse and read files in this workspace
- `cave_activate` - Mark this workspace as active
- `cave_deactivate` - Mark this workspace as idle when done

## Important Notes

- This workspace is isolated from other agent workspaces
- Changes here do not affect other agents' work
- The workspace path is managed by AgentCave - do not modify `.agentcave/` files directly
