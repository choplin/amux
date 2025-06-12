package mcp

import (
	"testing"
)

func TestBridgeToolsRegistration(t *testing.T) {
	// Just verify that setupTestServer succeeds, which includes registering all bridge tools
	testServer := setupTestServer(t)

	if testServer == nil {
		t.Fatal("expected server to be created with all bridge tools registered")
	}
}
