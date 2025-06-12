package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestPromptBridgeTools(t *testing.T) {
	testServer := setupTestServer(t)

	t.Run("prompt_list returns available prompts", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "prompt_list",
				Arguments: map[string]interface{}{},
			},
		}

		result, err := testServer.handlePromptList(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content, got %d", len(result.Content))
		}

		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", result.Content[0])
		}

		var prompts []map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &prompts); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Should have at least some prompts registered
		if len(prompts) == 0 {
			t.Error("expected at least one prompt to be registered")
		}

		// Check structure
		for _, prompt := range prompts {
			if _, ok := prompt["name"]; !ok {
				t.Error("prompt missing 'name' field")
			}
			if _, ok := prompt["description"]; !ok {
				t.Error("prompt missing 'description' field")
			}
		}
	})

	t.Run("prompt_get returns specific prompt", func(t *testing.T) {
		// First get the list to know what prompts are available
		listReq := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      "prompt_list",
				Arguments: map[string]interface{}{},
			},
		}

		listResult, err := testServer.handlePromptList(context.Background(), listReq)
		if err != nil {
			t.Fatalf("failed to get prompt list: %v", err)
		}

		textContent, _ := listResult.Content[0].(mcp.TextContent)
		var prompts []map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &prompts); err != nil {
			t.Fatalf("failed to unmarshal prompt list: %v", err)
		}

		if len(prompts) == 0 {
			t.Skip("no prompts available to test")
		}

		// Get the first prompt
		promptName := prompts[0]["name"].(string)

		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "prompt_get",
				Arguments: map[string]interface{}{
					"name": promptName,
				},
			},
		}

		result, err := testServer.handlePromptGet(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content, got %d", len(result.Content))
		}

		textContent, ok := result.Content[0].(mcp.TextContent)
		if !ok {
			t.Fatalf("expected TextContent, got %T", result.Content[0])
		}

		var detail map[string]interface{}
		if err := json.Unmarshal([]byte(textContent.Text), &detail); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if detail["name"] != promptName {
			t.Errorf("expected prompt name %s, got %v", promptName, detail["name"])
		}
		if _, ok := detail["description"]; !ok {
			t.Error("prompt missing 'description' field")
		}
	})

	t.Run("prompt_get returns error for non-existent prompt", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "prompt_get",
				Arguments: map[string]interface{}{
					"name": "non-existent-prompt",
				},
			},
		}

		_, err := testServer.handlePromptGet(context.Background(), req)
		if err == nil {
			t.Error("expected error for non-existent prompt, got nil")
		}
	})
}

func TestPromptBridgeToolsRegistration(t *testing.T) {
	testServer := setupTestServer(t)

	// Just verify that setupTestServer succeeds, which includes registering prompt bridge tools
	if testServer == nil {
		t.Fatal("expected server to be created with prompt bridge tools registered")
	}
}
