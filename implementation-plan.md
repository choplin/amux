# MCP Resources and Prompts Implementation Plan

## Overview

This document outlines the step-by-step implementation of the MCP Resources and Prompts architecture as described in ADR-009.

## Phase 1: Resource Infrastructure

### 1.1 Resource Handler Base

- [ ] Create `internal/mcp/resources/handler.go`
  - Resource interface matching MCP spec
  - URI parsing and routing
  - Error handling

### 1.2 Conventions Resource

- [ ] Implement `amux://conventions` resource
  - Static JSON response with paths and patterns
  - Version field for future compatibility
  - Test coverage

### 1.3 Workspace Resources

- [ ] Implement `amux://workspace` (list all)
- [ ] Implement `amux://workspace/{id}` (get details)
- [ ] Implement `amux://workspace/{id}/files[/{path}]` (browse files)
- [ ] Implement `amux://workspace/{id}/context` (read context.md)

### 1.4 Resource Registration

- [ ] Update MCP server to register resources
- [ ] Add resource discovery endpoint
- [ ] Update server initialization

## Phase 2: Prompt Infrastructure

### 2.1 Prompt Handler Base

- [ ] Create `internal/mcp/prompts/handler.go`
  - Prompt interface matching MCP spec
  - Argument validation
  - Template rendering

### 2.2 Core Prompts

- [ ] Implement `start-issue-work` prompt
  - Emphasis on understanding requirements
  - Interactive clarification steps
  - Context.md template generation

- [ ] Implement `prepare-pr-submission` prompt
  - Workspace validation
  - Test execution guidance
  - PR template generation

- [ ] Implement `manage-multiple-tasks` prompt
  - Workspace listing integration
  - Best practices guidance

- [ ] Implement `ai-agent-collaboration` prompt
  - Mailbox usage patterns
  - Context sharing guidelines

### 2.3 Prompt Registration

- [ ] Update MCP server to register prompts
- [ ] Add prompt discovery endpoint
- [ ] Test prompt execution flow

## Phase 3: Tool Refactoring

### 3.1 Deprecate workspace_info

- [ ] Add deprecation notice to workspace_info
- [ ] Update tool description with migration guide
- [ ] Log deprecation warnings

### 3.2 Clean up Tool Descriptions

- [ ] Add clear descriptions to all tools
- [ ] Specify which resource to use for read operations
- [ ] Update tool metadata

## Phase 4: Testing

### 4.1 Unit Tests

- [ ] Resource handler tests
- [ ] URI parsing tests
- [ ] Prompt validation tests
- [ ] Conventions resource tests

### 4.2 Integration Tests

- [ ] Full MCP server with resources
- [ ] Resource discovery flow
- [ ] Prompt execution flow
- [ ] Backwards compatibility tests

## Phase 5: Documentation

### 5.1 Update CLAUDE.md

- [ ] Document new resource URIs
- [ ] Explain prompt usage
- [ ] Migration guide from tools to resources

### 5.2 Update README.md

- [ ] Add MCP resources section
- [ ] Document available prompts
- [ ] Update examples

### 5.3 Create Migration Guide

- [ ] workspace_info → resources migration
- [ ] Example code updates
- [ ] Timeline for deprecation

## Implementation Order

1. Start with conventions resource (simplest)
2. Add workspace list resource
3. Add workspace detail resources
4. Implement start-issue-work prompt
5. Test with real AI agent
6. Implement remaining resources and prompts
7. Deprecate workspace_info
8. Update all documentation

## Success Metrics

- [ ] AI agent can discover all resources via list
- [ ] Conventions resource provides all necessary paths
- [ ] File browsing works through resources
- [ ] Prompts guide through complete workflows
- [ ] Existing tool users get clear migration path
- [ ] No breaking changes for 2 releases

## Notes

- Keep resource responses lightweight
- Use proper HTTP-style error codes in responses
- Consider caching for static resources
- Ensure prompts are idempotent where possible

- Test with multiple AI providers (Claude, GPT, etc.)
