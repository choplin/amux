package context

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// Context directory name within workspace
	ContextDir = ".amux/context"

	// Context file names
	BackgroundFile     = "background.md"
	PlanFile           = "plan.md"
	WorkingLogFile     = "working-log.md"
	ResultsSummaryFile = "results-summary.md"
)

// Manager handles working context for AI agents
type Manager struct {
	workspacePath string
	contextPath   string
}

// NewManager creates a new context manager for a workspace
func NewManager(workspacePath string) *Manager {
	contextPath := filepath.Join(workspacePath, ContextDir)
	return &Manager{
		workspacePath: workspacePath,
		contextPath:   contextPath,
	}
}

// Initialize creates the context directory and template files
func (m *Manager) Initialize() error {
	// Create context directory
	if err := os.MkdirAll(m.contextPath, 0755); err != nil {
		return fmt.Errorf("failed to create context directory: %w", err)
	}

	// Initialize template files
	if err := m.initializeBackground(); err != nil {
		return err
	}
	if err := m.initializePlan(); err != nil {
		return err
	}
	if err := m.initializeWorkingLog(); err != nil {
		return err
	}
	if err := m.initializeResultsSummary(); err != nil {
		return err
	}

	return nil
}

// initializeBackground creates the background.md template
func (m *Manager) initializeBackground() error {
	filePath := filepath.Join(m.contextPath, BackgroundFile)

	// Don't overwrite existing file
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}

	content := `# Background

## Task Overview
<!-- Brief description of what needs to be accomplished -->

## Requirements
<!-- Specific requirements and constraints -->

## Context
<!-- Any relevant project context, dependencies, or considerations -->

## Success Criteria
<!-- How we'll know when the task is complete -->
`

	return os.WriteFile(filePath, []byte(content), 0644)
}

// initializePlan creates the plan.md template
func (m *Manager) initializePlan() error {
	filePath := filepath.Join(m.contextPath, PlanFile)

	// Don't overwrite existing file
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}

	content := `# Implementation Plan

## Approach
<!-- High-level approach to solving the task -->

## Steps
<!-- Detailed steps to implement the solution -->

1.
2.
3.

## Technical Decisions
<!-- Key technical decisions and rationale -->

## Risks & Mitigations
<!-- Potential issues and how to handle them -->
`

	return os.WriteFile(filePath, []byte(content), 0644)
}

// initializeWorkingLog creates the working-log.md template
func (m *Manager) initializeWorkingLog() error {
	filePath := filepath.Join(m.contextPath, WorkingLogFile)

	// Don't overwrite existing file
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}

	content := fmt.Sprintf(`# Working Log

## Session Started: %s

### Progress
<!-- Update this section as you work -->

### Decisions Made
<!-- Document key decisions and reasoning -->

### Issues Encountered
<!-- Track any problems and solutions -->
`, time.Now().Format("2006-01-02 15:04:05"))

	return os.WriteFile(filePath, []byte(content), 0644)
}

// initializeResultsSummary creates the results-summary.md template
func (m *Manager) initializeResultsSummary() error {
	filePath := filepath.Join(m.contextPath, ResultsSummaryFile)

	// Don't overwrite existing file
	if _, err := os.Stat(filePath); err == nil {
		return nil
	}

	content := `# Results Summary

## Overview
<!-- Brief summary of what was accomplished -->

## Changes Made
<!-- List of key changes -->

-
-
-

## Testing
<!-- How the changes were tested -->

## Next Steps
<!-- Any follow-up work needed -->

## Notes for Review
<!-- Important information for code review -->
`

	return os.WriteFile(filePath, []byte(content), 0644)
}

// GetContextPath returns the full path to the context directory
func (m *Manager) GetContextPath() string {
	return m.contextPath
}

// GetFilePath returns the full path to a context file
func (m *Manager) GetFilePath(filename string) string {
	return filepath.Join(m.contextPath, filename)
}

// Exists checks if the context directory exists
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.contextPath)
	return err == nil
}

// AppendToWorkingLog adds an entry to the working log
func (m *Manager) AppendToWorkingLog(entry string) error {
	filePath := filepath.Join(m.contextPath, WorkingLogFile)

	// Open file in append mode
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open working log: %w", err)
	}
	defer file.Close()

	// Add timestamp and entry
	timestamp := time.Now().Format("15:04:05")
	content := fmt.Sprintf("\n### [%s] %s\n", timestamp, entry)

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to working log: %w", err)
	}

	return nil

}
