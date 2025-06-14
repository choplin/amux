# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed

- **BREAKING**: Removed mailbox feature in favor of generalized storage directories
  - Session mailbox system has been deprecated
  - Workspaces now have a general-purpose `storage/` directory
  - Sessions now have a general-purpose `storage/` directory
  - MCP tools `storage_read`, `storage_write`, and `storage_list` added for storage access

### Changed

- Workspace context is now stored in the storage directory (`storage/context.md`)
- Session and workspace metadata now include storage paths

## [0.1.0] - 2025-06-13

### Added

- Initial release of amux (Agent Multiplexer)
- Core workspace management functionality
  - Create isolated git worktree-based workspaces
  - List all workspaces with short numeric IDs
  - Show detailed workspace information
  - Remove workspaces safely
  - Prune old workspaces by age
- Git integration
  - Each workspace is a separate git worktree
  - Support for creating workspaces from existing branches
  - Automatic branch creation for new workspaces
- MCP (Model Context Protocol) server integration
  - Full integration with Claude Code
  - Workspace management through MCP tools
  - Resource browsing capabilities
- Session mailbox system for agent communication (CLI only)
- Version command showing version, git commit, build date, and system info
- Comprehensive test suite with >80% coverage
- Pre-commit hooks for code quality
- Documentation including architecture decisions (ADRs)

### Security

- Workspaces are isolated from each other
- No automatic commits or pushes
- Safe workspace removal with validation

[unreleased]: https://github.com/choplin/amux/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/choplin/amux/releases/tag/v0.1.0
