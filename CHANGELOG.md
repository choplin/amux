# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-06-16

### Added

- Workspace management - Create, list, navigate, and manage isolated git worktree-based workspaces
- AI agent session management - Run agents with `amux run`, monitor with `amux ps`, attach to sessions
- MCP (Model Context Protocol) server for Claude Code integration
- Configuration management with JSON Schema validation
- Real-time session status tracking (busy/idle/stuck)
- Session log streaming with `amux tail` and `amux session logs -f`
- Storage directories for workspaces and sessions
- Comprehensive test suite with >80% coverage
- Pre-commit hooks for code quality
[0.1.0]: https://github.com/choplin/amux/releases/tag/v0.1.0
