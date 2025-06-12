# MCP Schema Utilities

This package provides utility functions to simplify working with the mcp-go library by using Go struct tags to define
MCP tool schemas.

## Features

- **`StructToToolOptions`** - Convert Go structs to `[]mcp.ToolOption`
- **`WithStructOptions`** - Helper that combines description with struct options
- **`UnmarshalArgs`** - Type-safe unmarshaling of `CallToolRequest` arguments

## Usage

### 1. Define your parameter struct with tags

```go
type MyToolParams struct {
    // Required fields use mcp:"required"
    Name string `json:"name" mcp:"required" description:"Item name"`

    // Optional fields omit the mcp tag
    Count int `json:"count,omitempty" description:"Number of items"`

    // Enum values can be specified
    Status string `json:"status,omitempty" description:"Status" enum:"\"active\",\"inactive\""`
}
```

### 2. Create tool with struct options

```go
// Simple approach using WithStructOptions
opts, _ := WithStructOptions("My tool description", MyToolParams{})
tool := mcp.NewTool("my_tool", opts...)

// Or use StructToToolOptions for more control
baseOpts, _ := StructToToolOptions(MyToolParams{})
tool := mcp.NewTool("my_tool",
    append([]mcp.ToolOption{
        mcp.WithDescription("My tool"),
        // other options...
    }, baseOpts...)...,
)
```

### 3. Unmarshal arguments in handlers

```go
func handleMyTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    var params MyToolParams
    if err := UnmarshalArgs(request, &params); err != nil {
        return nil, err
    }

    // Use params with full type safety
    fmt.Printf("Name: %s, Count: %d\n", params.Name, params.Count)
    // ...
}
```

## Supported Types

- `string` - Maps to `mcp.WithString()`
- `int`, `int64` - Maps to `mcp.WithNumber()`
- `bool` - Maps to `mcp.WithBoolean()`
- Arrays/slices - Not yet supported (mcp-go limitation)

## Tag Reference

- `json:"fieldname"` - JSON field name (required)
- `mcp:"required"` - Mark field as required
- `description:"text"` - Field description
- `enum:"\"val1\",\"val2\""` - Enum values (string fields only)
