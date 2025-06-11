// Package mcp provides Model Context Protocol server implementation for Amux.
package mcp

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/mark3labs/mcp-go/mcp"
)

// StructToToolOptions converts a Go struct to mcp-go tool options using reflection

// StructToToolOptions converts a struct with tags into MCP tool options.
// The struct should use tags like `json:"name" mcp:"required" description:"Workspace name"`
func StructToToolOptions(structType interface{}) ([]mcp.ToolOption, error) {
	t := reflect.TypeOf(structType)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct type, got %v", t.Kind())
	}

	// Start building the tool options

	var toolOptions []mcp.ToolOption

	// Process each field

	for i := 0; i < t.NumField(); i++ {

		field := t.Field(i)

		// Get JSON tag for field name

		jsonTag := field.Tag.Get("json")

		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Extract field name from json tag (handle "name,omitempty")

		fieldName := jsonTag

		if idx := len(jsonTag); idx > 0 {
			for j := 0; j < len(jsonTag); j++ {
				if jsonTag[j] == ',' {

					fieldName = jsonTag[:j]

					break

				}
			}
		}

		// Get description from tag

		description := field.Tag.Get("description")

		if description == "" {
			description = fmt.Sprintf("%s field", fieldName)
		}

		// Check if required

		mcpTag := field.Tag.Get("mcp")

		isRequired := mcpTag == "required"

		// Get enum values if specified

		enumTag := field.Tag.Get("enum")

		// Add field based on type

		switch field.Type.Kind() { //nolint:exhaustive // Only handling types we support

		case reflect.String:

			opts := []mcp.PropertyOption{
				mcp.Description(description),
			}

			if isRequired {
				opts = append(opts, mcp.Required())
			}

			if enumTag != "" {

				// Parse enum values (comma-separated)

				var enumValues []string

				if err := json.Unmarshal([]byte("["+enumTag+"]"), &enumValues); err == nil {
					opts = append(opts, mcp.Enum(enumValues...))
				}

			}

			toolOptions = append(toolOptions, mcp.WithString(fieldName, opts...))

		case reflect.Int, reflect.Int64:

			opts := []mcp.PropertyOption{
				mcp.Description(description),
			}

			if isRequired {
				opts = append(opts, mcp.Required())
			}

			toolOptions = append(toolOptions, mcp.WithNumber(fieldName, opts...))

		case reflect.Bool:

			opts := []mcp.PropertyOption{
				mcp.Description(description),
			}

			if isRequired {
				opts = append(opts, mcp.Required())
			}

			toolOptions = append(toolOptions, mcp.WithBoolean(fieldName, opts...))

		case reflect.Slice:
			// TODO: Add support for array types when available in mcp-go
			// Currently, array properties are not supported in mcp-go
			_ = isRequired // Suppress unused variable warning

		default:

			// Skip unsupported types

			continue

		}

	}

	return toolOptions, nil
}

// WithStructOptions is a helper that combines a description with struct-based options
func WithStructOptions(description string, structType interface{}) ([]mcp.ToolOption, error) {
	structOpts, err := StructToToolOptions(structType)
	if err != nil {
		return nil, err
	}

	// Prepend the description

	return append([]mcp.ToolOption{mcp.WithDescription(description)}, structOpts...), nil
}

// UnmarshalArgs unmarshals CallToolRequest arguments into a struct
func UnmarshalArgs[T any](request mcp.CallToolRequest, target *T) error {
	args := request.GetArguments()

	// Convert map to JSON then unmarshal to struct

	jsonBytes, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("failed to marshal arguments: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal arguments to struct: %w", err)
	}

	return nil
}

// WorkspaceCreateParams defines parameters for creating a workspace
type WorkspaceCreateParams struct {
	Name string `json:"name" mcp:"required" description:"Workspace name"`

	BaseBranch string `json:"baseBranch,omitempty" description:"Base branch (optional)"`

	Branch string `json:"branch,omitempty" description:"Use existing branch (optional)"`

	AgentID string `json:"agentId,omitempty" description:"Agent ID (optional)"`

	Description string `json:"description,omitempty" description:"Description (optional)"`
}

// WorkspaceListParams defines parameters for listing workspaces
type WorkspaceListParams struct {
	// No parameters needed for listing workspaces
}

// WorkspaceIDParams defines parameters for workspace operations requiring an ID
type WorkspaceIDParams struct {
	WorkspaceID string `json:"workspace_id" mcp:"required" description:"Workspace name or ID"`
}

// WorkspaceInfoParams defines parameters for getting workspace file information
type WorkspaceInfoParams struct {
	WorkspaceID string `json:"workspace_id" mcp:"required" description:"Workspace name or ID"`

	Path string `json:"path,omitempty" description:"File or directory path (optional)"`
}
