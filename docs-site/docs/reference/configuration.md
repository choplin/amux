---
sidebar_position: 2
---

# Configuration Reference

Complete guide to Amux configuration with JSON Schema validation.

## Configuration File Format v1.0

Amux uses `.amux/config.yaml` with strict JSON Schema validation:

```yaml
version: "1.0"
project:
  name: my-project
  repository: https://github.com/user/my-project.git
  defaultAgent: claude

mcp:
  transport:
    type: stdio  # or "http"

agents:
  claude:
    name: Claude
    type: tmux  # Required field
    description: Claude AI assistant
    params:  # Type-specific parameters
      command: claude
      shell: /bin/bash
```

## JSON Schema Validation

Amux validates all configuration files against a JSON Schema. This provides:

- **Type safety** - Catch configuration errors before runtime
- **Auto-completion** - IDE support with schema references
- **Clear error messages** - Know exactly what's wrong

### Schema Location

The full schema is available at:

- Built into Amux: `amux config schema`
- GitHub: [config.schema.json](https://github.com/choplin/amux/blob/main/internal/core/config/schemas/config.schema.json)

### VS Code Integration

Add to your workspace settings (`.vscode/settings.json`):

```json
{
  "yaml.schemas": {
    "https://raw.githubusercontent.com/choplin/amux/main/internal/core/config/schemas/config.schema.json": ".amux/config.yaml"
  }
}
```

## Configuration Commands

### View Configuration

```bash
# Show current configuration (YAML format)
amux config show

# Output as JSON
amux config show --format json

# Pretty-printed with syntax highlighting
amux config show --format pretty
```

### Edit Configuration

```bash
# Open in default editor
amux config edit

# Use specific editor
EDITOR=vim amux config edit
```

### Validate Configuration

```bash
# Validate configuration file
amux config validate

# Show detailed validation errors
amux config validate --verbose
```

### Show Schema

```bash
# Output JSON Schema
amux config schema

# Save schema to file
amux config schema > config.schema.json
```

## Configuration Structure

### Project Section

```yaml
project:
  name: my-project         # Required: Project identifier
  repository: https://...  # Optional: Git repository URL
  defaultAgent: claude     # Optional: Default agent ID
```

### MCP Transport Configuration

```yaml
mcp:
  transport:
    type: stdio  # Currently only stdio is supported
```

### Agent Configuration

Each agent requires:

- **name**: Display name
- **type**: Session type (currently only `tmux`)
- **params**: Type-specific parameters

```yaml
agents:
  claude:
    name: Claude
    type: tmux
    description: Claude AI assistant  # Optional
    environment:                      # Optional
      API_KEY: ${CLAUDE_API_KEY}
    workingDir: /path/to/dir         # Optional
    tags: [ai, assistant]            # Optional
    params:
      command: claude
      shell: /bin/bash               # Optional
      windowName: claude-session     # Optional
      detached: false                # Optional
```

## Complete Configuration Examples

### Basic Configuration

```yaml
version: "1.0"
project:
  name: my-app
agents:
  claude:
    name: Claude
    type: tmux
    params:
      command: claude
```

### Advanced Configuration with Multiple Agents

```yaml
version: "1.0"
project:
  name: enterprise-app
  repository: https://github.com/company/app.git
  defaultAgent: claude

mcp:
  transport:
    type: stdio

agents:
  claude:
    name: Claude AI
    type: tmux
    description: Primary development assistant
    environment:
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
      PROJECT_ENV: development
    tags: [primary, ai, development]
    params:
      command: claude
      shell: /bin/zsh
      windowName: claude-dev

  aider:
    name: Aider
    type: tmux
    description: Code editing assistant
    params:
      command: aider
      detached: true
```

## Environment Variables

Amux supports environment variable substitution in configuration:

```yaml
agents:
  claude:
    environment:
      API_KEY: ${CLAUDE_API_KEY}        # From environment
      DEFAULT: ${VAR:-default_value}    # With default value
```

## Validation and Troubleshooting

### Common Validation Errors

```bash
# Missing required field
Error: agents.claude: missing required field 'type'

# Invalid enum value
Error: mcp.transport.type: must be one of ["stdio", "http"]

# Type mismatch
Error: mcp.transport.http.port: expected integer, got string
```

### Debugging Configuration

```bash
# Check what Amux sees after parsing
amux config show --format json

# Validate without running
amux config validate --verbose

# Test specific agent configuration
amux agent validate claude
```

## Best Practices

1. **Start simple**: Begin with minimal configuration and add features as needed
2. **Use environment variables**: Keep sensitive data out of the config file
3. **Validate often**: Run `amux config validate` after changes
4. **Use schema**: Enable IDE integration for auto-completion
5. **Document agents**: Use the `description` field to explain each agent's purpose
