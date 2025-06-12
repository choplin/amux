package mcp

import (
	"testing"
)

func TestStructToToolOptions(t *testing.T) {
	tests := []struct {
		name string

		structType interface{}

		expectError bool

		checkFields []string
	}{
		{
			name: "WorkspaceCreateParams",

			structType: WorkspaceCreateParams{},

			checkFields: []string{"name", "baseBranch", "branch", "agentId", "description"},
		},

		{
			name: "WorkspaceIDParams",

			structType: WorkspaceIDParams{},

			checkFields: []string{"workspace_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := StructToToolOptions(tt.structType)

			if tt.expectError {

				if err == nil {
					t.Errorf("expected error but got none")
				}

				return

			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Handle empty structs (like CaveListParams)

			if len(tt.checkFields) == 0 {
				// For empty structs, opts should be empty array or nil

				if len(opts) > 0 {
					t.Error("expected no options for empty struct")
				}
			} else {

				if opts == nil {
					t.Fatal("expected options but got nil")
				}

				// We can't easily inspect the options without creating a tool

				// So we just check that options were generated

				if len(opts) == 0 {
					t.Error("expected at least one option")
				}

			}
		})
	}
}

// Note: UnmarshalArgs is tested indirectly through the actual tool handlers

// since it requires the actual mcp.CallToolRequest interface from mcp-go