# 23. JSON Schema Validation for Configuration

Date: 2025-06-16

## Status

Accepted

## Context

Amux uses YAML configuration files to define project settings, MCP configuration, and agent definitions. Previously, we used custom Go code to validate these configurations, which had several limitations:

1. **Maintenance burden**: Every new field or validation rule required updating Go code
2. **Limited error messages**: Custom validation provided basic error messages without detailed context
3. **No editor support**: Users couldn't get autocompletion or inline validation in their editors
4. **Inconsistent validation**: Different parts of the codebase might validate the same fields differently
5. **Documentation drift**: Validation rules might not match the documented configuration format

As we prepare to support multiple session types beyond tmux (e.g., `claude-code`, `api`, `lsp`), the configuration structure is becoming more complex with type-specific validation requirements.

## Decision

We will use JSON Schema validation for all configuration validation, replacing the custom Go validation code.

Implementation details:

- Use `github.com/santhosh-tekuri/jsonschema/v5` library for validation
- Define schema in `internal/core/config/schemas/config.schema.json`
- Validate YAML content before unmarshaling to Go structs
- Embed the schema in the binary using Go's `embed` package
- Support JSON Schema draft 2020-12 for modern features

The validation flow:

1. Read YAML file
2. Unmarshal YAML to generic `interface{}`
3. Validate against JSON Schema
4. If valid, unmarshal to typed Go structs
5. Apply defaults

## Consequences

### Positive

- **Standardization**: JSON Schema is an industry standard, well-understood by developers
- **Better error messages**: Schema validation provides detailed paths and reasons for failures
- **Editor support**: Users can reference the schema in VS Code and other editors for autocompletion
- **Single source of truth**: The schema serves as both validation and documentation
- **Easier maintenance**: Adding new fields only requires updating the schema
- **Reusability**: The schema can be used for documentation generation, API validation, etc.
- **Type-specific validation**: Easy to define different requirements for different agent types using `if/then` conditions

### Negative

- **Additional dependency**: Adds a new library dependency to the project
- **Schema complexity**: JSON Schema can be verbose and complex for advanced validation rules
- **Two-step validation**: We still need Go struct tags for unmarshaling after schema validation
- **Learning curve**: Contributors need to understand JSON Schema syntax

### Neutral

- Configuration loading performance remains similar (schema compilation is cached)
- Error messages change format, which may affect scripts parsing them
- The `amux config validate` command behavior remains the same from the user's perspective

## Implementation Notes

The schema defines:

- Required fields at each level
- Enum constraints for version and type fields
- Pattern constraints for agent IDs
- Conditional requirements (e.g., tmux agents must have tmux configuration)
- Additional properties restrictions to catch typos

### Important: Type-Specific Field Validation

The schema uses a unified `params` field for all type-specific configurations, avoiding field name conflicts between different agent types. The `params` field content is validated based on the agent's `type` field value.

For example:

- `tmux` agents require `params` to contain tmux-specific fields (command, shell, etc.)
- Future `claude-code` agents will require different fields in `params`

This is implemented using conditional validation:

```json
"if": {
  "properties": { "type": { "const": "tmux" } }
},
"then": {
  "properties": {
    "params": {
      "type": "object",
      "required": ["command"],
      "properties": {
        "command": { "type": "string" },
        "shell": { "type": "string" },
        "windowName": { "type": "string" },
        "detached": { "type": "boolean" }
      }
    }
  }
}
```

Future agent types can be added by:

1. Adding the type to the `agent.type` enum
2. Adding a new conditional validation rule for the `params` content
3. Implementing the corresponding Go type and getter method
4. The unified `params` field prevents naming conflicts between types
