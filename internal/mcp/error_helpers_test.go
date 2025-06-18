package mcp

import (
	"strings"
	"testing"
)

func TestErrorWithSuggestions(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantMessage string
		wantSuggest []string
	}{
		{
			name:        "workspace not found",
			err:         WorkspaceNotFoundError("test-ws"),
			wantMessage: "workspace not found: test-ws",
			wantSuggest: []string{
				"resource_workspace_list",
				"workspace_create",
			},
		},
		{
			name:        "session not found",
			err:         SessionNotFoundError("session-123"),
			wantMessage: "session not found: session-123",
			wantSuggest: []string{
				"resource_session_list",
				"session_run",
			},
		},
		{
			name:        "no workspaces",
			err:         NoWorkspacesError(),
			wantMessage: "no workspaces found",
			wantSuggest: []string{
				"workspace_create",
				"resource_workspace_list",
			},
		},
		{
			name:        "file not found",
			err:         FileNotFoundError("test.txt"),
			wantMessage: "file not found: test.txt",
			wantSuggest: []string{
				"resource_workspace_browse",
				"workspace_storage_list",
				"workspace_storage_write",
			},
		},
		{
			name:        "directory not found",
			err:         DirectoryNotFoundError("test-dir"),
			wantMessage: "directory not found: test-dir",
			wantSuggest: []string{
				"resource_workspace_browse",
				"workspace_storage_list",
			},
		},
		{
			name:        "session not running",
			err:         SessionNotRunningError("session-456"),
			wantMessage: "session session-456 is not running",
			wantSuggest: []string{
				"session_run",
				"resource_session_list",
				"resource_session_output",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()

			// Check message
			if !strings.Contains(errStr, tt.wantMessage) {
				t.Errorf("error message should contain %q, got %q", tt.wantMessage, errStr)
			}

			// Check suggestions
			for _, suggest := range tt.wantSuggest {
				if !strings.Contains(errStr, suggest) {
					t.Errorf("error should suggest %q, but it's missing", suggest)
				}
			}

			// Verify format includes suggestion header
			if len(tt.wantSuggest) > 0 && !strings.Contains(errStr, "Did you mean to use one of these tools instead?") {
				t.Error("error with suggestions should include suggestion header")
			}
		})
	}
}

func TestNewErrorWithSuggestions(t *testing.T) {
	err := NewErrorWithSuggestions("custom error", "tool1", "tool2")
	errStr := err.Error()

	if !strings.Contains(errStr, "custom error") {
		t.Error("error should contain custom message")
	}
	if !strings.Contains(errStr, "tool1") {
		t.Error("error should contain tool1 suggestion")
	}
	if !strings.Contains(errStr, "tool2") {
		t.Error("error should contain tool2 suggestion")
	}
}

func TestErrorWithoutSuggestions(t *testing.T) {
	err := NewErrorWithSuggestions("simple error")
	if err.Error() != "simple error" {
		t.Errorf("error without suggestions should return simple message, got %q", err.Error())
	}
}
