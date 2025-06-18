package mcp

import (
	"fmt"
	"strings"
)

// ErrorWithSuggestions represents an error with tool suggestions
type ErrorWithSuggestions struct {
	Message     string
	Suggestions []string
}

// Error returns the error message with suggestions
func (e *ErrorWithSuggestions) Error() string {
	if len(e.Suggestions) == 0 {
		return e.Message
	}

	var sb strings.Builder
	sb.WriteString(e.Message)
	sb.WriteString("\n\nDid you mean to use one of these tools instead?\n")
	for _, suggestion := range e.Suggestions {
		sb.WriteString("  - ")
		sb.WriteString(suggestion)
		sb.WriteString("\n")
	}
	return sb.String()
}

// NewErrorWithSuggestions creates a new error with tool suggestions
func NewErrorWithSuggestions(message string, suggestions ...string) error {
	return &ErrorWithSuggestions{
		Message:     message,
		Suggestions: suggestions,
	}
}

// WorkspaceNotFoundError returns an error with suggestions for when a workspace is not found
func WorkspaceNotFoundError(identifier string) error {
	return NewErrorWithSuggestions(
		fmt.Sprintf("workspace not found: %s", identifier),
		"resource_workspace_list - List all available workspaces",
		"workspace_create - Create a new workspace",
	)
}

// SessionNotFoundError returns an error with suggestions for when a session is not found
func SessionNotFoundError(identifier string) error {
	return NewErrorWithSuggestions(
		fmt.Sprintf("session not found: %s", identifier),
		"resource_session_list - List all active sessions",
		"session_run - Start a new session",
	)
}

// NoWorkspacesError returns an error with suggestions for when no workspaces exist
func NoWorkspacesError() error {
	return NewErrorWithSuggestions(
		"no workspaces found",
		"workspace_create - Create your first workspace",
		"resource_workspace_list - Verify workspace status",
	)
}

// FileNotFoundError returns an error with suggestions for when a file is not found
func FileNotFoundError(path string) error {
	return NewErrorWithSuggestions(
		fmt.Sprintf("file not found: %s", path),
		"resource_workspace_browse - Browse workspace files",
		"workspace_storage_list - List files in workspace storage",
		"workspace_storage_write - Create the file in workspace storage",
	)
}

// DirectoryNotFoundError returns an error with suggestions for when a directory is not found
func DirectoryNotFoundError(path string) error {
	return NewErrorWithSuggestions(
		fmt.Sprintf("directory not found: %s", path),
		"resource_workspace_browse - Browse workspace structure",
		"workspace_storage_list - List workspace storage contents",
	)
}

// SessionNotRunningError returns an error with suggestions for when a session is not running
func SessionNotRunningError(identifier string) error {
	return NewErrorWithSuggestions(
		fmt.Sprintf("session %s is not running", identifier),
		"session_run - Start a new session",
		"resource_session_list - Check session status",
		"resource_session_output - View session output",
	)
}

// InvalidParameterError returns an error with suggestions for invalid parameters
func InvalidParameterError(param string, expected string) error {
	return NewErrorWithSuggestions(
		fmt.Sprintf("invalid %s: expected %s", param, expected),
		"Use the tool descriptions to understand parameter requirements",
		"Check examples in the tool description",
	)
}
