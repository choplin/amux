---
sidebar_position: 4
---

# Lifecycle Hooks

Amux provides lifecycle hooks to automatically execute commands at key points in the workspace and session lifecycle. This enables powerful automation for environment setup, dependency installation, and cleanup tasks.

## Overview

Hooks allow you to:

- Install dependencies when creating workspaces
- Set up environment files and configurations
- Start background services with sessions
- Clean up resources when workspaces are removed
- Automate any repetitive setup tasks

## Hook Events

Amux supports four lifecycle events:

### Workspace Hooks

- **`workspace_create`** - Runs after a workspace is created
- **`workspace_remove`** - Runs before a workspace is removed

### Session Hooks

- **`session_start`** - Runs after a session starts
- **`session_stop`** - Runs before a session stops

## Execution Directory

Hooks execute in context-appropriate directories:

- **Workspace hooks** execute in the workspace directory
- **Session hooks** execute in the session's assigned workspace directory

This means you can run commands like `npm install` or `poetry install` directly without needing to change directories.

## Configuration

Hooks are configured in `.amux/config.yaml`:

```yaml
version: "1.0"

hooks:
  workspace_create:
    - name: "Install dependencies"
      command: "npm install"
    - name: "Setup environment"
      command: "cp .env.example .env"

  workspace_remove:
    - name: "Clean up temp files"
      command: "rm -rf tmp/*"

  session_start:
    - name: "Start database"
      command: "docker-compose up -d db"

  session_stop:
    - name: "Stop database"
      command: "docker-compose down"
```

### Hook Properties

Each hook entry requires:

- **`name`** - A descriptive name for the hook
- **`command`** - The command to execute

Commands are executed directly without shell interpretation. If you need shell features (pipes, redirections), explicitly invoke a shell:

```yaml
hooks:
  workspace_create:
    - name: "Create file with redirection"
      command: "sh -c 'echo test > output.txt'"
    - name: "Use pipes"
      command: "sh -c 'cat input.txt | grep pattern'"
```

## Environment Variables

Hooks have access to Amux-specific environment variables:

- **`AMUX_PROJECT_ROOT`** - Path to the project root directory
- **`AMUX_WORKSPACE_PATH`** - Path to the current workspace
- **`AMUX_WORKSPACE_ID`** - Workspace identifier
- **`AMUX_WORKSPACE_NAME`** - Workspace name
- **`AMUX_SESSION_ID`** - Session ID (session hooks only)
- **`AMUX_AGENT_ID`** - Agent identifier (session hooks only)

## Common Use Cases

### JavaScript/Node.js Projects

```yaml
hooks:
  workspace_create:
    - name: "Install dependencies"
      command: "npm install"
    - name: "Build project"
      command: "npm run build"
    - name: "Run tests"
      command: "npm test"
```

### Python Projects

```yaml
hooks:
  workspace_create:
    - name: "Create virtual environment"
      command: "python -m venv .venv"
    - name: "Install dependencies"
      command: "sh -c '.venv/bin/pip install -r requirements.txt'"
    - name: "Install dev dependencies"
      command: "sh -c '.venv/bin/pip install -r requirements-dev.txt'"
```

### Go Projects

```yaml
hooks:
  workspace_create:
    - name: "Download dependencies"
      command: "go mod download"
    - name: "Run tests"
      command: "go test ./..."
    - name: "Build binary"
      command: "go build -o bin/app"
```

### Docker-based Development

```yaml
hooks:
  session_start:
    - name: "Start services"
      command: "docker-compose up -d"
    - name: "Wait for database"
      command: "sh -c 'until docker-compose exec -T db pg_isready; do sleep 1; done'"

  session_stop:
    - name: "Stop services"
      command: "docker-compose down"
```

### Environment Setup

```yaml
hooks:
  workspace_create:
    - name: "Copy environment template"
      command: "cp .env.example .env"
    - name: "Generate secret key"
      command: "sh -c 'echo SECRET_KEY=$(openssl rand -hex 32) >> .env'"
    - name: "Create directories"
      command: "mkdir -p tmp logs uploads"
```

## Error Handling

- If a hook fails, subsequent hooks in the same event will not execute
- Hook failures are reported but don't prevent the main operation (workspace creation, session start, etc.)
- Exit codes are logged for debugging

## Best Practices

### 1. Keep Hooks Fast

Hooks run synchronously, so keep them quick:

```yaml
# Good - Fast setup
hooks:
  workspace_create:
    - name: "Quick setup"
      command: "cp .env.example .env"

# Consider running heavy tasks in background
hooks:
  session_start:
    - name: "Start slow service in background"
      command: "sh -c 'docker-compose up -d heavy-service &'"
```

### 2. Make Hooks Idempotent

Hooks should be safe to run multiple times:

```yaml
# Good - Creates directory only if it doesn't exist
hooks:
  workspace_create:
    - name: "Ensure directories exist"
      command: "mkdir -p tmp logs"

# Bad - Fails if directory already exists
hooks:
  workspace_create:
    - name: "Create directory"
      command: "mkdir tmp"
```

### 3. Use Project Root for Shared Resources

When hooks need to access project-level resources:

```yaml
hooks:
  workspace_create:
    - name: "Run project script"
      command: "sh -c 'cd $AMUX_PROJECT_ROOT && ./scripts/setup.sh $AMUX_WORKSPACE_PATH'"
```

### 4. Handle Platform Differences

Consider cross-platform compatibility:

```yaml
hooks:
  workspace_create:
    - name: "Platform-specific setup"
      command: "sh -c 'if [ -f /etc/debian_version ]; then apt-get update; elif [ -f /etc/redhat-release ]; then yum update; fi'"
```

## Debugging Hooks

### View Hook Output

Hook output is displayed when running commands:

```bash
amux ws create feature-auth
# Shows hook execution and output
```

### Test Hooks Manually

You can test hook commands in a workspace:

```bash
# Create workspace without hooks
amux ws create test --no-hooks

# Test commands manually
cd .amux/workspaces/workspace-test-*/worktree
npm install  # Test what the hook would do
```

### Common Issues

1. **Command not found**: Ensure the command is in PATH or use absolute paths
2. **Permission denied**: Check file permissions and execution rights
3. **Working directory**: Remember hooks run in the workspace directory
4. **Shell features**: Use `sh -c` for pipes, redirections, and shell syntax

## Advanced Examples

### Conditional Execution

```yaml
hooks:
  workspace_create:
    - name: "Install deps if package.json exists"
      command: "sh -c '[ -f package.json ] && npm install || true'"
```

### Multiple Steps with Error Handling

```yaml
hooks:
  workspace_create:
    - name: "Full setup with checks"
      command: "sh -c 'npm install && npm run build && npm test || echo Setup failed'"
```

### Using External Scripts

```yaml
hooks:
  workspace_create:
    - name: "Run setup script"
      command: "sh -c 'chmod +x setup.sh && ./setup.sh'"
```

## Migration from Root Directory Execution

Prior to the hooks revision, all hooks executed in the project root. If you have existing hooks that depend on this behavior, update them to use `$AMUX_PROJECT_ROOT`:

```yaml
# Old behavior (executed in project root)
hooks:
  workspace_create:
    - name: "Old hook"
      command: "./scripts/setup.sh"

# New behavior (executed in workspace)
hooks:
  workspace_create:
    - name: "Updated hook"
      command: "sh -c 'cd $AMUX_PROJECT_ROOT && ./scripts/setup.sh'"
```
