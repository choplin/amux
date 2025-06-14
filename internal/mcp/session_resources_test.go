package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestSessionResources(t *testing.T) {
	// Create test server
	testServer := setupTestServer(t)

	t.Run("session list resource returns empty list initially", func(t *testing.T) {
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: "amux://session",
			},
		}

		results, err := testServer.handleSessionListResource(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}

		textResource, ok := results[0].(*mcp.TextResourceContents)
		if !ok {
			t.Fatalf("expected TextResourceContents, got %T", results[0])
		}

		// Parse JSON response
		var sessions []sessionInfo
		if err := json.Unmarshal([]byte(textResource.Text), &sessions); err != nil {
			t.Fatalf("failed to parse session list: %v", err)
		}

		if len(sessions) != 0 {
			t.Errorf("expected empty session list, got %d sessions", len(sessions))
		}
	})

	t.Run("session detail resource returns error for non-existent session", func(t *testing.T) {
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: "amux://session/non-existent",
			},
		}

		_, err := testServer.handleSessionDetailResource(context.Background(), req)
		if err == nil {
			t.Error("expected error for non-existent session, got nil")
		}
	})

	t.Run("session output resource returns error for non-existent session", func(t *testing.T) {
		req := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: "amux://session/non-existent/output",
			},
		}

		_, err := testServer.handleSessionOutputResource(context.Background(), req)
		if err == nil {
			t.Error("expected error for non-existent session, got nil")
		}
	})
}

func TestSessionResourceRegistration(t *testing.T) {
	// Create test server
	testServer := setupTestServer(t)

	// Test that resources can be registered without error
	err := testServer.registerSessionResources()
	if err != nil {
		t.Fatalf("failed to register session resources: %v", err)
	}

	// Verify resources are registered by checking the MCP server
	// Note: This is a basic test - more comprehensive tests would require
	// creating actual sessions and verifying the resource outputs
}
