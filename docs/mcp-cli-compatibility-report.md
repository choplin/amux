# MCP-CLI Compatibility Report

This document analyzes the compatibility between MCP tools and CLI commands in amux.

## Overview

The analysis shows that MCP tools and CLI commands use **consistent parameter naming** with the `_identifier` suffix pattern for workspace and session references. This ensures compatibility and clarity.

## Workspace Management

### Parameter Naming Consistency

Both MCP tools and CLI commands use the same identifier pattern:

| Component | Parameter Name | Type | Description |
|-----------|---------------|------|-------------|
| MCP Tools | `workspace_identifier` | string | Workspace ID, index, or name |
| CLI Commands | First positional argument | string | Workspace ID, index, or name |

### Workspace Operations

#### Create Workspace

- **CLI**: `amux ws create <name> [--description "..."] [--branch existing-branch]`
- **MCP**: `workspace_create({ name, description?, branch?, baseBranch? })`
- **Compatibility**: ✅ Full compatibility with matching parameters

#### List Workspaces

- **CLI**: `amux ws list [--oneline]`
- **MCP**: `resource_workspace_list()`
- **Compatibility**: ✅ Full compatibility

#### Show Workspace

- **CLI**: `amux ws show <workspace-name-or-id>`
- **MCP**: `resource_workspace_show({ workspace_identifier })`
- **Compatibility**: ✅ Full compatibility

#### Remove Workspace

- **CLI**: `amux ws remove <workspace-name-or-id> [--force]`
- **MCP**: `workspace_remove({ workspace_identifier })`
- **Compatibility**: ✅ Full compatibility

#### Browse Workspace Files

- **CLI**: No direct equivalent (use standard file tools)
- **MCP**: `resource_workspace_browse({ workspace_identifier, path? })`
- **Compatibility**: ✅ MCP-specific feature for remote file access

## Session Management

### Session Parameter Naming

Both MCP tools and CLI commands use the same identifier pattern:

| Component | Parameter Name | Type | Description |
|-----------|---------------|------|-------------|
| MCP Tools | `session_identifier` | string | Session ID, index, or name |
| CLI Commands | First positional argument | string | Session ID, index, or name |

### Session Operations

#### Run Session

- **CLI**: `amux session run <agent> [--workspace <id>] [--name "..."] [--description "..."]`
- **MCP**: `session_run({ workspace_identifier, agent_id, name?, description?, command? })`
- **Compatibility**: ✅ Full compatibility with matching parameters

#### Stop Session

- **CLI**: `amux session stop <session>`
- **MCP**: `session_stop({ session_identifier })`
- **Compatibility**: ✅ Full compatibility

#### List Sessions

- **CLI**: `amux session list` (aliases: `ls`, `ps`)
- **MCP**: `resource_session_list()`
- **Compatibility**: ✅ Full compatibility

#### Send Input to Session

- **CLI**: `amux session send-input <session> --input "..."`
- **MCP**: `session_send_input({ session_identifier, input })`
- **Compatibility**: ✅ Full compatibility

#### Session Resources

- **CLI**: No direct equivalent
- **MCP**: `resource_session_show({ session_id })`, `resource_session_output({ session_id })`
- **Compatibility**: ✅ MCP-specific features for remote access

## Storage Operations

### Storage Parameter Naming

Storage operations maintain the same identifier pattern:

| Component | Parameter Names | Type | Description |
|-----------|----------------|------|-------------|
| MCP Tools | `workspace_identifier` OR `session_identifier` | string | Target storage location |
| CLI Commands | No direct storage commands | - | Storage is handled via file system |

### Storage Operations (MCP-only)

- `storage_read({ workspace_identifier?, session_identifier?, path })`
- `storage_write({ workspace_identifier?, session_identifier?, path, content })`
- `storage_list({ workspace_identifier?, session_identifier?, path? })`

**Note**: These are MCP-specific tools for remote storage access. CLI users interact with storage directly through the file system.

## Key Findings

### 1. Consistent Identifier Naming ✅

All MCP tools and CLI commands use the `_identifier` suffix pattern:

- `workspace_identifier` for workspace references
- `session_identifier` for session references

This naming convention clearly indicates that these parameters accept multiple forms of identification (ID, index, or name).

### 2. Parameter Mapping

CLI positional arguments map directly to MCP tool parameters:

- CLI: `amux ws show <workspace-name-or-id>`
- MCP: `resource_workspace_show({ workspace_identifier: "..." })`

### 3. Feature Parity

Most operations have direct equivalents between CLI and MCP:

- Workspace: create, list, show, remove
- Session: run, stop, list, send-input

### 4. MCP-Specific Features

Some features are MCP-only for remote access:

- File browsing (`resource_workspace_browse`)
- Session output streaming (`resource_session_output`)
- Storage operations (`storage_*` tools)

### 5. Resolution Logic

Both CLI and MCP use the same resolution logic for identifiers:

1. Try exact ID match
2. Try numeric index match
3. Try name match
4. Return error if not found

## Recommendations

1. **Documentation**: Update documentation to emphasize the `_identifier` suffix pattern
2. **Consistency**: Maintain this naming pattern for any new features
3. **Testing**: Ensure resolution logic remains consistent between CLI and MCP
4. **User Experience**: Consider adding examples showing how the same identifier works in both CLI and MCP

## Conclusion

The MCP tools and CLI commands demonstrate excellent compatibility with consistent parameter naming using the `_identifier` suffix pattern. This design choice makes it clear to users that these parameters accept flexible identification formats (ID, index, or name), ensuring a seamless experience whether using the CLI directly or through MCP integration.
