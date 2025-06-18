# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-06-16

### Added

- Workspace management - Create, list, navigate, and manage isolated git worktree-based workspaces
- AI agent session management - Run agents with `amux run`, monitor with `amux ps`, attach to sessions
- MCP (Model Context Protocol) server for Claude Code integration
- Tmux parameters support - Custom shell, window name, and environment variables for sessions
- Auto-attach support - Automatically attach to tmux sessions after creation
- Session and workspace hooks - Execute custom commands on session/workspace lifecycle events
- Configuration management with JSON Schema validation
- Real-time session status tracking (busy/idle/stuck)
- Session log streaming with `amux tail` and `amux session logs -f`
- Separate MCP storage endpoints for workspaces and sessions
- Storage directories for workspaces and sessions
- Interactive .gitignore updates with user consent
- Comprehensive test suite with >80% coverage
- Pre-commit hooks for code quality
[0.1.0]: <https://github.com/choplin/amux/releases/tag/v0.1.0>
