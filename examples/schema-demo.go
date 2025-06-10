// Example demonstrating the schema utility functions for mcp-go
package main

import (
	"fmt"
	"log"

	mcpgo "github.com/mark3labs/mcp-go/mcp"

	"github.com/aki/amux/internal/mcp"
)

// Define your tool parameters as structs with proper tags
type MyToolParams struct {
	// Required fields use mcp:"required" tag
	Name string `json:"name" mcp:"required" description:"The name of the item"`

	// Optional fields don't have the mcp tag
	Count int `json:"count,omitempty" description:"Number of items (optional)"`

	// Enums can be specified with the enum tag
	Status string `json:"status,omitempty" description:"Current status" enum:"\"active\",\"inactive\",\"pending\""`

	// Boolean fields
	Enabled bool `json:"enabled,omitempty" description:"Whether the feature is enabled"`
}

func main() {
	// Method 1: Using WithStructOptions (recommended for most cases)
	toolOptions, err := mcp.WithStructOptions("Example tool demonstrating schema conversion", MyToolParams{})
	if err != nil {
		log.Fatalf("Failed to create tool options: %v", err)
	}

	tool := mcpgo.NewTool("my_tool", toolOptions...)
	fmt.Printf("Created tool: %s\n", tool.Name)
	fmt.Printf("Description: %s\n", tool.Description)

	// Method 2: Using StructToToolOptions (when you need more control)
	baseOptions, err := mcp.StructToToolOptions(MyToolParams{})
	if err != nil {
		log.Fatalf("Failed to create tool options: %v", err)
	}

	// You can prepend or append additional options
	customTool := mcpgo.NewTool("custom_tool",
		append([]mcpgo.ToolOption{
			mcpgo.WithDescription("Custom tool with extra options"),
			// Add any other custom options here
		}, baseOptions...)...,
	)
	fmt.Printf("\nCreated custom tool: %s\n", customTool.Name)

	// In your actual MCP handler, you would use UnmarshalArgs like this:
	// func handleMyTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	//     var params MyToolParams
	//     if err := mcp.UnmarshalArgs(request, &params); err != nil {
	//         return nil, fmt.Errorf("failed to unmarshal arguments: %w", err)
	//     }
	//
	//     // Now you can use params.Name, params.Count, etc. with full type safety
	//     fmt.Printf("Processing: %s (count: %d)\n", params.Name, params.Count)
	//
	//     return &mcp.CallToolResult{
	//         Content: []mcp.Content{
	//             mcp.TextContent{
	//                 Type: "text",
	//                 Text: fmt.Sprintf("Processed %s successfully", params.Name),
	//             },
	//         },
	//     }, nil
	// }

	fmt.Println("\nBenefits of this approach:")
	fmt.Println("1. Type-safe parameter handling - no more manual type assertions")
	fmt.Println("2. Single source of truth - struct tags define both schema and parsing")
	fmt.Println("3. Compile-time safety - typos in field names are caught at compile time")
	fmt.Println("4. Less boilerplate - no need to manually build tool definitions")
	fmt.Println("5. Automatic validation - required fields are enforced by mcp-go")
}
