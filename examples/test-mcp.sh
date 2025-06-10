#!/bin/bash
# Test script for Amux MCP server

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Testing Amux MCP Server${NC}"
echo "================================"

# Function to send JSON-RPC request
send_request() {
    local request=$1
    echo -e "\n${BLUE}Request:${NC}"
    echo "$request" | jq .
    echo -e "\n${BLUE}Response:${NC}"
    echo "$request" | go run cmd/amux/main.go serve --transport stdio 2>/dev/null | jq .
}

# Test 1: Initialize
echo -e "\n${GREEN}Test 1: Initialize${NC}"
send_request '{
  "jsonrpc": "2.0",
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "test-client",
      "version": "1.0.0"
    }
  },
  "id": 1
}'

# Test 2: List Tools
echo -e "\n${GREEN}Test 2: List Tools${NC}"
send_request '{
  "jsonrpc": "2.0",
  "method": "tools/list",
  "id": 2
}'

# Test 3: Create Cave
echo -e "\n${GREEN}Test 3: Create Cave${NC}"
send_request '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "cave_create",
    "arguments": {
      "name": "test-workspace",
      "description": "Test workspace for MCP testing"
    }
  },
  "id": 3
}'

# Test 4: List Caves
echo -e "\n${GREEN}Test 4: List Caves${NC}"
send_request '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "cave_list",
    "arguments": {}
  },
  "id": 4
}'

echo -e "\n${BLUE}MCP Server Tests Complete${NC}"