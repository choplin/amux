package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolResultMetadata contains metadata for tool results
type ToolResultMetadata struct {
	ToolUsed           string              `json:"tool_used"`
	InferredParameters map[string]string   `json:"inferred_parameters,omitempty"`
	SuggestedNextTools []map[string]string `json:"suggested_next_tools,omitempty"`
}

// createEnhancedResult creates a tool result with metadata
func createEnhancedResult(toolName string, content interface{}, metadata *ToolResultMetadata) (*mcp.CallToolResult, error) {
	// Prepare the enhanced result
	type EnhancedResult struct {
		Result   interface{}         `json:"result"`
		Metadata *ToolResultMetadata `json:"_metadata,omitempty"`
	}

	enhanced := EnhancedResult{
		Result: content,
	}

	// Add metadata if provided
	if metadata == nil {
		metadata = &ToolResultMetadata{
			ToolUsed: toolName,
		}
	} else {
		metadata.ToolUsed = toolName
	}

	// Add suggested next tools
	metadata.SuggestedNextTools = GetNextToolSuggestions(toolName)
	enhanced.Metadata = metadata

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(enhanced, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
	}, nil
}
