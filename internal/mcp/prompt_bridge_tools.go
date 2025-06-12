package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// PromptGetParams contains parameters for prompt_get tool
type PromptGetParams struct {
	Name string `json:"name" jsonschema:"required,description=Name of the prompt to retrieve"`
}

// registerPromptBridgeTools registers bridge tools for prompt resources
func (s *ServerV2) registerPromptBridgeTools() error {
	// prompt_list - List available prompts
	promptListOpts, err := WithStructOptions(
		"List all available prompts. Returns prompt names and descriptions.",
		struct{}{},
	)
	if err != nil {
		return fmt.Errorf("failed to create prompt_list options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("prompt_list", promptListOpts...), s.handlePromptList)

	// prompt_get - Get a specific prompt
	promptGetOpts, err := WithStructOptions(
		"Get a specific prompt by name. Returns the prompt definition including description and arguments.",
		PromptGetParams{},
	)
	if err != nil {
		return fmt.Errorf("failed to create prompt_get options: %w", err)
	}
	s.mcpServer.AddTool(mcp.NewTool("prompt_get", promptGetOpts...), s.handlePromptGet)

	return nil
}

// Bridge tool handlers for prompt resources

func (s *ServerV2) handlePromptList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Use the shared getPromptList logic
	promptList, err := s.getPromptList()
	if err != nil {
		return nil, err
	}

	jsonData, err := json.MarshalIndent(promptList, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prompt list: %w", err)
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

func (s *ServerV2) handlePromptGet(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Use the shared getPromptDetail logic
	detail, err := s.getPromptDetail(name)
	if err != nil {
		return nil, err
	}

	jsonData, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal prompt detail: %w", err)
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
