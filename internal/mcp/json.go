package mcp

import (
	"encoding/json"
	"fmt"
)

// JSON-RPC 2.0 types

type Request struct {
	Jsonrpc string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

type Response struct {
	Jsonrpc string         `json:"jsonrpc"`
	Result  interface{}    `json:"result,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
	ID      interface{}    `json:"id"`
}

type ErrorResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// MCP-specific types

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

type ClientCapabilities struct {
	// Add capabilities as needed
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type ServerCapabilities struct {
	Tools ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type CallToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Helper functions

func NewResponse(id interface{}, result interface{}) *Response {
	return &Response{
		Jsonrpc: "2.0",
		Result:  result,
		ID:      id,
	}
}

func NewErrorResponse(id interface{}, code int, message string, data interface{}) *Response {
	return &Response{
		Jsonrpc: "2.0",
		Error: &ErrorResponse{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}
}

func NewToolContent(text string) []ToolContent {
	return []ToolContent{
		{
			Type: "text",
			Text: text,
		},
	}
}

func NewToolError(err error) *CallToolResult {
	return &CallToolResult{
		Content: NewToolContent(fmt.Sprintf("Error: %v", err)),
		IsError: true,
	}
}
