# ADR-027: Hook Execution Directory

**Status**: Accepted

## Context

Currently, all hooks in amux execute in the project root directory, regardless of which workspace triggered them. This severely limits the usefulness of hooks, as workspace-specific setup operations (like installing dependencies or setting up environment) cannot be performed without manual workarounds.

### Current Implementation Problems

1. **Manual Directory Changes**: Users must prefix every hook command with `cd $AMUX_WORKSPACE_PATH &&` to run commands in the correct directory
2. **Error-Prone**: Forgetting the cd prefix causes hooks to run in the wrong directory
3. **Verbose Configuration**: Every hook needs boilerplate for directory navigation
4. **Limited Usefulness**: Can't use simple commands like `npm install` or `poetry install` directly

### Example of Current Workaround

```yaml
hooks:
  workspace_create:
    - name: "Install dependencies"
      command: "cd $AMUX_WORKSPACE_PATH && npm install"
    - name: "Setup environment"
      command: "cd $AMUX_WORKSPACE_PATH && ./scripts/setup.sh"
```

## Decision

Change hook execution behavior to use context-appropriate working directories:

1. **Workspace hooks** (`workspace_create`, `workspace_remove`) - Execute in the workspace directory
2. **Session hooks** (`session_start`, `session_stop`) - Execute in the session's assigned workspace directory
3. **Session hooks without workspace** - Fail with clear error message

Additionally:
- Rename `agent_start/stop` events to `session_start/stop` for consistency with the rest of the codebase
- Execute commands through shell (`sh -c`) to support redirections, pipes, and other shell features

## Rationale

### Principle of Least Surprise

Hooks should run where the action occurred. When creating a workspace, it's natural to expect hooks to run in that workspace.

### Common Use Cases

Most hook use cases involve workspace-specific operations:
- Installing dependencies (`npm install`, `pip install`, `go mod download`)
- Setting up environment files (`.env`, `config.yaml`)
- Running initialization scripts
- Creating workspace-specific directories

### Consistency with Sessions

Sessions already run in their assigned workspaces. Hooks should follow the same pattern.

### Shell Execution

Using `sh -c` allows natural shell syntax:
- Redirections: `echo "test" > file.txt`
- Pipes: `cat file | grep pattern`
- Multiple commands: `command1 && command2`

## Consequences

### Positive

- Hooks become immediately useful for workspace setup
- No need for manual `cd` commands in hook definitions
- More intuitive behavior - hooks run where the action occurred
- Enables common use cases like dependency installation
- Consistent with session execution model

### Negative

- **Breaking change** for existing hooks that assume root directory execution
- Hooks that need to access project-level resources must use `$AMUX_PROJECT_ROOT`

### Migration Path

Existing hooks that rely on root directory execution must be updated:

```yaml
# Before
hooks:
  workspace_create:
    - name: "Run project script"
      command: "./scripts/setup.sh"

# After
hooks:
  workspace_create:
    - name: "Run project script"
      command: "cd $AMUX_PROJECT_ROOT && ./scripts/setup.sh"
```

## Implementation Notes

The implementation adds a `WithWorkingDir()` method to the hook executor:

```go
// Set working directory based on context
executor := hooks.NewExecutor(configDir, env).WithWorkingDir(workspace.Path)
```

For session hooks, workspace assignment is required:

```go
if workspace == nil {
    return fmt.Errorf("session hooks require workspace assignment")
}
```

## References

- Issue #167: Hooks execute in wrong directory
- Issue #169: Replace agent_start/stop with session_start/stop and implement session hooks
