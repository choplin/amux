# ADR-014: YAML for Configuration and Metadata

Date: 2025-06-12

## Status

Accepted

## Context

Amux needs a format for storing:

- Project configuration (`config.yaml`)
- Workspace metadata
- Session information
- Agent definitions

Requirements:

- Human-readable and editable
- Support for complex, nested structures
- Comments for documentation
- Wide language support
- Standard in DevOps/cloud-native tools

Options considered:

1. **JSON** - JavaScript Object Notation
2. **TOML** - Tom's Obvious, Minimal Language
3. **YAML** - YAML Ain't Markup Language
4. **HCL** - HashiCorp Configuration Language
5. **INI** - Simple key-value format

## Decision

We will use YAML as the primary format for all configuration and metadata files.

This includes:

- `.amux/config.yaml` - Project configuration
- `.amux/workspaces/workspace-*.yaml` - Workspace metadata
- `.amux/sessions/session-*.yaml` - Session information
- Embedded workspace metadata in `workspace.yaml`

## Consequences

### Positive

- **Human-readable** - Easy to read and edit manually
- **Comments** - Can document configuration inline
- **Complex structures** - Supports nested objects and arrays naturally
- **DevOps standard** - Familiar to users of Kubernetes, Docker Compose, etc.
- **Good library support** - Mature parsing libraries in all languages
- **Expressive** - Supports multi-line strings, references, etc.

### Negative

- **Parsing complexity** - YAML spec is complex, edge cases exist
- **Indentation sensitivity** - Errors from incorrect indentation
- **Performance** - Slower parsing than JSON
- **Security concerns** - Some YAML parsers have had vulnerabilities
- **Type ambiguity** - Strings vs numbers vs booleans can be unclear

### Neutral

- Requires YAML parsing library
- Schema validation needs additional tooling
- Different YAML versions have different features

## Implementation Notes

1. Use YAML 1.2 specification
2. Avoid complex YAML features (anchors, tags) for simplicity
3. Always quote strings that could be ambiguous
4. Use consistent indentation (2 spaces)
5. Validate schemas where possible
6. Be careful with type conversions

Example structure:

```yaml
# Clear comments explaining configuration
amux:
  version: 0.1.0
  root_dir: .amux

agents:
  claude:
    name: Claude
    type: claude
    command: claude
    environment:
      # Use ${} for environment variable substitution
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
```

## Alternatives Considered

### JSON

**Pros**: Fast parsing, unambiguous, wide support
**Cons**: No comments, verbose, not human-friendly, no multiline strings

### TOML

**Pros**: Human-friendly, clear types, good for flat configs
**Cons**: Complex nested structures are awkward, less common in DevOps

### HCL

**Pros**: Good for infrastructure config, supports functions
**Cons**: HashiCorp-specific, learning curve, overkill for our needs

### INI

**Pros**: Very simple, human-readable
**Cons**: No nested structures, limited types, too simple for our needs

## Migration Strategy

If we need to migrate away from YAML:

1. Implement new parser alongside YAML
2. Add migration command
3. Support both formats temporarily
4. Deprecate YAML support
5. Remove YAML parser

## References

- [YAML 1.2 Specification](https://yaml.org/spec/1.2/spec.html)
- [YAML Best Practices](https://yaml.org/spec/1.2/spec.html#id2805071)
- Go YAML v3 library documentation
