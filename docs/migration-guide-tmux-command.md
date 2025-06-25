# Tmux Agent Configuration Migration Guide

This guide explains how to migrate your tmux agent configuration from the old format to the new simplified format introduced in #218.

## Breaking Changes

### 1. Removed `shell` Field

The `shell` field has been removed from tmux agent configuration. The functionality is now handled through the `command` field.

**Before:**

```yaml
agents:
  myagent:
    type: tmux
    params:
      command: claude
      shell: /bin/zsh  # This field is removed
```

**After:**

```yaml
agents:
  myagent:
    type: tmux
    params:
      command: claude  # Runs in default shell
```

### 2. New Command Format

The `command` field now supports both string and array formats:

#### String Format (Shell Execution)

When `command` is a string, it's executed through the default shell:

```yaml
agents:
  myagent:
    type: tmux
    params:
      command: "echo 'Hello World'"  # Executed as: sh -c "echo 'Hello World'"
```

#### Array Format (Direct Execution)

When `command` is an array, it's executed directly without shell interpretation:

```yaml
agents:
  python-server:
    type: tmux
    params:
      command: ["python", "-m", "http.server", "8080"]  # Executed directly
```

## Migration Examples

### Example 1: Custom Shell

If you were using a custom shell:

**Before:**

```yaml
agents:
  zsh-agent:
    type: tmux
    params:
      shell: /bin/zsh
```

**After:**

```yaml
agents:
  zsh-agent:
    type: tmux
    params:
      command: /bin/zsh
```

### Example 2: Shell with Command

If you had both shell and command:

**Before:**

```yaml
agents:
  dev-server:
    type: tmux
    params:
      command: npm run dev
      shell: /bin/bash
```

**After:**

```yaml
agents:
  dev-server:
    type: tmux
    params:
      command: npm run dev  # Uses default shell
```

Or if you need a specific shell:

```yaml
agents:
  dev-server:
    type: tmux
    params:
      command: "/bin/bash -c 'npm run dev'"
```

### Example 3: Complex Commands

For commands with special characters or multiple arguments:

**Before (might have issues with escaping):**

```yaml
agents:
  complex:
    type: tmux
    params:
      command: 'find . -name "*.js" -exec echo {} \;'
```

**After (using array format for clarity):**

```yaml
agents:
  complex:
    type: tmux
    params:
      command: ["find", ".", "-name", "*.js", "-exec", "echo", "{}", ";"]
```

## Benefits of the New Format

1. **Simplicity**: One field instead of two reduces confusion
2. **Flexibility**: Array format allows precise command specification without shell escaping issues
3. **Consistency**: Follows the same pattern as Docker, Kubernetes, and other tools
4. **Clarity**: Makes it explicit whether shell interpretation is happening

## Notes

- If no command is specified, an error will be returned (no default shell fallback)
- The default shell is determined by the system (typically `/bin/sh`)
- Environment variables are still expanded in string commands (through shell)
- Array commands bypass shell interpretation entirely
