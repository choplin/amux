# 24. MCP Tool Discoverability

Date: 2025-06-16

## Status

Accepted

## Context

AI agents configured to use Amux MCP tools often struggle to discover and understand when to use our tools without explicit user instructions. This results in:

- AI agents not utilizing available tools even when they would be helpful
- Users having to repeatedly prompt AI agents to use specific tools
- AI agents misusing tools due to unclear documentation
- Poor workflow understanding, leading to inefficient tool usage patterns

The goal is to enable AI agents to automatically discover and use Amux tools whenever applicable, without requiring additional user action.

## Decision

We will enhance MCP tool discoverability through the following improvements:

### 1. Enhanced Tool Descriptions

Add comprehensive descriptions to all MCP tools including:
- Clear, detailed descriptions of what each tool does
- "WHEN TO USE THIS TOOL" sections with specific triggers and use cases
- Practical examples showing correct parameter usage
- Suggested next tools for common workflows

### 2. Tool Chaining Hints

Implement response metadata that includes:
- The tool that was used
- Suggested next tools based on common workflows
- Contextual hints about what actions typically follow

### 3. Improved Error Messages

Create custom error types that:
- Provide clear error descriptions
- Suggest alternative tools when appropriate
- Include actionable recovery steps

### 4. Explicit Parameter Requirements

Based on user preference for explicit behavior:
- Do NOT implement smart defaults or parameter inference
- Require all necessary parameters to be explicitly provided
- Keep tool behavior predictable and transparent

## Consequences

### Positive

- **Better AI Agent Autonomy**: AI agents can discover and use tools without user prompting
- **Improved Workflow Understanding**: Tool chaining hints guide AI agents through common patterns
- **Reduced User Friction**: Users don't need to repeatedly instruct AI agents on tool usage
- **Error Recovery**: AI agents can recover from errors with actionable suggestions
- **Backward Compatibility**: Enhanced descriptions are additive and don't break existing integrations

### Negative

- **Increased Payload Size**: Response metadata adds overhead to tool responses
- **Maintenance Burden**: Tool descriptions and suggestions need to be kept up-to-date
- **No Smart Defaults**: Users must always provide explicit parameters (by design)

### Implementation Notes

- Tool descriptions are centralized in `tool_descriptions.go` for easy maintenance
- Enhanced result format includes a `__metadata` field to avoid conflicts with content
- Error suggestions are contextual and help guide users to the right tool
- All changes are backward compatible with existing MCP clients
