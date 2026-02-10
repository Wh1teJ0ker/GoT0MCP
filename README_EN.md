# GoT0MCP: High-Performance MCP Server Library for Go

[English](README.md) | [中文](README_CN.md)

GoT0MCP is a Model Context Protocol (MCP) server library designed specifically for Go. Inspired by Python's `FastMCP`, it aims to provide a minimalist development experience while fully leveraging Go's high concurrency, strong typing, and high performance.

## Core Features

- **🚀 Rapid Development**: Similar to `FastMCP`, it automatically registers Go functions as MCP tools via reflection and generates JSON Schemas automatically.
- **⚡️ High Performance Architecture**: Uses `goccy/go-json` instead of the standard library for ultimate JSON serialization/deserialization performance. Coupled with a `Server` and `Session` separation architecture, it natively supports high concurrency.
- **🛡️ Robust Design**: Built-in timeout control, Panic capture, and recovery mechanisms ensure stable service operation.
- **📝 Structured Logging**: Integrated with `log/slog`, providing detailed structured logs for easy debugging and production monitoring.
- **🔌 Dual Transport Protocols**: Out-of-the-box support for Stdio (for local LLM/IDE integration) and WebSocket (for remote services).

## Documentation

- [Getting Started](docs/en/getting-started.md)
- [Architecture Overview](docs/en/architecture.md)

## Quick Preview

### 1. Installation

```bash
go get github.com/Wh1teJ0ker/GoT0MCP
```

### 2. Create a Simple MCP Server

```go
package main

import (
	"context"
	"fmt"
	"github.com/Wh1teJ0ker/GoT0MCP/pkg/mcp"
)

// Define argument struct, supporting JSON tags and jsonschema descriptions
type AddArgs struct {
	A int `json:"a" jsonschema:"description=First number"`
	B int `json:"b" jsonschema:"description=Second number"`
}

func main() {
	// 1. Create Server
	s := mcp.NewServer()

	// 2. Register Tool
	// Pass the function directly; the library automatically parses argument types and generates Schema
	s.AddTool("add", "Calculate sum of two numbers", func(ctx context.Context, args AddArgs) (int, error) {
		return args.A + args.B, nil
	})

	// 3. Start Service (Default Stdio)
	if err := s.ServeStdio(); err != nil {
		panic(err)
	}
}
```

## License

MIT License
