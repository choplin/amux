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
amux agent list
```

Output:

```text
SESSION ID           AGENT      WORKSPACE            STATUS     RUNTIME
session-abc123       claude     feature-auth         running    5m30s
session-def456       gpt        bugfix-api          running    2m15s
session-ghi789       gemini     docs-update         stopped    10m45s
```

### Attaching to Sessions

Attach to a running agent session:

```bash
amux attach session-abc123
# or
amux agent attach session-abc123
```

To detach from a tmux session without stopping it, use `Ctrl-B D`.

### Stopping Sessions

Stop a running session:

```bash
amux agent stop session-abc123
```

**Important**: Sessions in Amux are one-shot. Once stopped, a session cannot be resumed or restarted. To continue
work in the same workspace, start a new session with `amux run`. The workspace files and working context are
preserved, allowing agents to pick up where they left off.

### Viewing Session Output

View the current output of a session:

```bash
amux agent logs session-abc123
```

## Agent Configuration

### Configuring Agents

Add or update agent configurations:

```bash
# Add a new agent
amux agent config add gpt-4 \
  --name "GPT-4" \
  --type openai \
  --command "gpt-cli" \
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

Agents are configured in `.amux/config.yaml`:

```yaml
agents:
  claude:
    name: Claude
    type: claude
    command: claude
    environment:
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
  gpt:
    name: GPT-4
    type: openai
    command: gpt-cli
    environment:
      OPENAI_API_KEY: ${OPENAI_API_KEY}
```

## Working Context

Each workspace automatically gets working context files to help AI agents maintain context:

### Context Files

Located in `.amux/context/` within each workspace:

- **background.md** - Task requirements and constraints
- **plan.md** - Implementation approach and breakdown
- **working-log.md** - Progress tracking with timestamps
- **results-summary.md** - Final outcomes for review

### Managing Context

```bash
# Show context file paths for a workspace
amux ws context show [workspace]

# Initialize context files manually
amux ws context init [workspace]
```

### Environment Variable

The context path is automatically available in agent sessions:

```bash
echo $AMUX_CONTEXT_PATH
# Output: /path/to/workspace/.amux/context
```

## Advanced Usage

### Multiple Concurrent Agents

Run different agents in separate workspaces:

```bash
# Terminal 1: Start Claude for feature development
amux ws create feature-oauth --agent claude
amux run claude --workspace feature-oauth

# Terminal 2: Start GPT for bug fixing
amux ws create bugfix-api --agent gpt
amux run gpt --workspace bugfix-api

# Terminal 3: Monitor all sessions
amux ps
```

### Session Environment

Each session automatically includes:

- `AMUX_WORKSPACE_ID` - Workspace identifier
- `AMUX_WORKSPACE_PATH` - Full path to workspace
- `AMUX_SESSION_ID` - Unique session identifier
- `AMUX_AGENT_ID` - Agent identifier
- `AMUX_CONTEXT_PATH` - Path to working context files
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
   # Check session status
   amux ps

   # View current output
   amux agent logs session-abc123

   # Attach if needed
   amux attach session-abc123
   ```

5. **Complete work**:

   ```bash
   # Stop session when done
   amux agent stop session-abc123

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
