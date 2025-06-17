# Agent Multiplexing Guide

Agent multiplexing allows you to run multiple AI agents concurrently in isolated workspaces, each with their own
terminal session, environment, and working context.

## Overview

Amux provides session management capabilities that enable:

- Running multiple AI agents simultaneously in different workspaces
- Isolated terminal sessions using tmux
- Persistent session state across reconnections
- Working context management for AI agents
- Environment variable management per agent

## Prerequisites

- Git repository initialized with Amux (`amux init`)
- tmux installed (optional but recommended for full functionality)

## Basic Usage

### Running an Agent

Start an AI agent in a workspace:

```bash
# Run Claude in the latest workspace
amux run claude

# Run an agent in a specific workspace
amux run claude --workspace feature-auth

# Run with custom command
amux run claude --command "claude code --model opus"

# Run with environment variables
amux run claude --env ANTHROPIC_API_KEY=sk-...
```

### Managing Sessions

List running agent sessions:

```bash
amux ps
# or
amux session list
# or
amux status
```

Output:

```text
SESSION ID           AGENT      WORKSPACE            STATUS     IN STATUS   TOTAL TIME
session-abc123       claude     feature-auth         working    45s         5m30s
session-def456       aider      bugfix-api          idle       2m 15s      8m45s
session-ghi789       my-agent   docs-update         stopped    5m          15m45s
```

The **IN STATUS** column shows how long the session has been in its current status,
while **TOTAL TIME** shows the total elapsed time since the session started.

### Attaching to Sessions

Attach to a running agent session:

```bash
amux attach session-abc123
# or
amux session attach session-abc123
```

To detach from a tmux session without stopping it, use `Ctrl-B D`.

### Stopping Sessions

Stop a running session:

```bash
amux session stop session-abc123
```

**Important**: Sessions in Amux are one-shot. Once stopped, a session cannot be resumed or restarted. To continue
work in the same workspace, start a new session with `amux run`. The workspace files and working context are
preserved, allowing agents to pick up where they left off.

### Viewing Session Output

View the current output of a session:

```bash
amux session logs session-abc123
```

## Agent Configuration

### Configuring Agents

Add or update agent configurations:

```bash
# Add a new agent
amux agent config add aider \
  --name "Aider" \
  --type tmux \
  --command "aider" \
  --env OPENAI_API_KEY=sk-...

# Update an existing agent
amux agent config update claude \
  --command "claude --model sonnet" \
  --env ANTHROPIC_API_KEY=sk-new...

# List configured agents
amux agent config list

# Show agent details
amux agent config show claude
```

### Configuration File

Agents are configured in `.amux/config.yaml`.

#### Tmux Parameters

For tmux-based agents, you can configure the following parameters:

- **command** (required): The command to execute when starting the agent
- **shell** (optional): Custom shell to use (defaults to system shell)
  - Examples: `/bin/bash`, `/bin/zsh`, `/bin/fish`
  - The specified shell will be used to run the agent command
- **windowName** (optional): Custom name for the tmux window
  - Helps identify sessions in tmux's window list
  - Defaults to the session name if not specified

Example configuration:

```yaml
# Optional: Add schema reference for VS Code IntelliSense
# $schema: https://github.com/choplin/amux/schemas/config.schema.json

agents:
  claude:
    name: Claude
    type: tmux  # Required: session type (currently only "tmux" is supported)
    description: Claude AI assistant for development
    environment:
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
    params:
      command: claude
      shell: /bin/bash  # Optional: custom shell (e.g., /bin/zsh, /bin/fish)
      windowName: claude-dev  # Optional: tmux window name
      autoAttach: false  # Optional: automatically attach to session after creation (CLI only)

  aider:
    name: Aider
    type: tmux
    description: AI pair programming assistant
    environment:
      OPENAI_API_KEY: ${OPENAI_API_KEY}
    params:
      command: aider

  my-agent:
    name: My Custom Agent
    type: tmux
    description: Custom AI agent
    environment:
      API_KEY: ${MY_AGENT_API_KEY}
      MODEL: gpt-4
    params:
      command: my-agent-cli --model ${MODEL}
      shell: /bin/zsh  # Use zsh for this agent
```

### Auto-Attach Feature

The `autoAttach` parameter allows agents to automatically attach to their tmux session after creation when running from the CLI:

```yaml
agents:
  claude-interactive:
    name: Claude Interactive
    type: tmux
    params:
      command: claude
      autoAttach: true  # Automatically attach when started from CLI
```

**Important Notes:**
- Auto-attach only works when running from a terminal with TTY support
- When running via MCP (from AI agents), auto-attach is ignored and attach commands are returned instead
- Use `Ctrl-B D` to detach from an auto-attached session without stopping it

**Use Cases:**
- Interactive debugging sessions
- Initial setup or configuration that requires user input
- Educational demonstrations
- Short interactive tasks

## Working Context

Each workspace automatically gets working context files to help AI agents maintain context:

### Context Files

Located in `.amux/context/` within each workspace:

- **background.md** - Task requirements and constraints
- **plan.md** - Implementation approach and breakdown
- **working-log.md** - Progress tracking with timestamps
- **results-summary.md** - Final outcomes for review

### Managing Context

Context files are managed through the workspace storage directory.

## Advanced Usage

### Multiple Concurrent Agents

Run different agents in separate workspaces:

```bash
# Terminal 1: Start Claude for feature development
amux ws create feature-oauth --agent claude
amux run claude --workspace feature-oauth

# Terminal 2: Start Aider for bug fixing
amux ws create bugfix-api --agent aider
amux run aider --workspace bugfix-api

# Terminal 3: Monitor all sessions
amux ps
```

### Session Environment

Each session automatically includes:

- `AMUX_WORKSPACE_ID` - Workspace identifier
- `AMUX_WORKSPACE_PATH` - Full path to workspace
- `AMUX_SESSION_ID` - Unique session identifier
- `AMUX_AGENT_ID` - Agent identifier
- Plus any agent-specific environment variables

### Workflow Example

1. **Create workspace with agent assignment**:

   ```bash
   amux ws create feature-auth --agent claude --description "Implement OAuth 2.0"
   ```

2. **Start agent session**:

   ```bash
   amux run claude
   ```

3. **Agent works autonomously**:
   - Reads task from `background.md`
   - Plans approach in `plan.md`
   - Updates progress in `working-log.md`
   - Summarizes results in `results-summary.md`

4. **Monitor progress**:

   ```bash
   # Check session status and how long it's been in that state
   amux ps
   # The IN STATUS column shows how long each session has been working/idle/stopped

   # View current output
   amux session logs session-abc123

   # Attach if needed (especially for long idle sessions)
   amux attach session-abc123
   ```

5. **Complete work**:

   ```bash
   # Stop session when done
   amux session stop session-abc123

   # Review results
   cat .amux/context/results-summary.md
   ```

## Troubleshooting

### tmux Not Available

If tmux is not installed, sessions will run in basic mode without terminal multiplexing. Install tmux for full functionality:

```bash
# macOS
brew install tmux

# Ubuntu/Debian
sudo apt-get install tmux

# RHEL/CentOS
sudo yum install tmux
```

### Session Won't Start

1. Check workspace exists: `amux ws list`
2. Verify agent configuration: `amux agent config show <agent>`
3. Check for error messages in session creation
4. Ensure tmux is installed and accessible

### Can't Attach to Session

1. Verify session is running: `amux ps`
2. Check you're using the correct session ID
3. Ensure tmux is available on your system

### Environment Variables Not Set

1. Check agent configuration includes the variables
2. Verify variables are exported in your shell for runtime expansion
3. Use `--env` flag to override at runtime

## Best Practices

1. **One Agent Per Workspace** - Keep agent work isolated
2. **Use Descriptive Names** - Make workspaces and sessions easy to identify
3. **Configure Default Commands** - Set up agents with their typical commands
4. **Monitor Active Sessions** - Regularly check `amux ps` for running sessions
5. **Clean Up Completed Work** - Stop sessions and remove workspaces when done
6. **Document in Context Files** - Encourage agents to use the working context

7. **Review Before Merging** - Check `results-summary.md` before creating PRs
