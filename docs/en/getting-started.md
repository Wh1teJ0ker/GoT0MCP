# Getting Started Guide

This document guides you through creating a GoT0MCP server from scratch and integrating it with Claude Desktop or other MCP clients.

## Prerequisites

- **Go**: 1.21 or higher
- **Git**: For downloading the repository

## 1. Installation

Run the following in your Go project:

```bash
go get github.com/Wh1teJ0ker/GoT0MCP
```

## 2. Write Your First MCP Server

Create a `main.go` file:

```go
package main

import (
	"context"
	"fmt"
	"github.com/Wh1teJ0ker/GoT0MCP/pkg/mcp"
)

// WeatherArgs defines arguments for weather query
type WeatherArgs struct {
	City string `json:"city" jsonschema:"description=The name of the city to query"`
}

func main() {
	// 1. Initialize Server
	s := mcp.NewServer(mcp.ServerOptions{
		Name: "WeatherServer",
		Version: "0.1.0",
	})

	// 2. Register Tool
	s.AddTool("get_weather", "Get weather for a specific city", func(ctx context.Context, args WeatherArgs) (string, error) {
		// This could be a real API call
		return fmt.Sprintf("The weather in %s is Sunny, 25°C", args.City), nil
	})

	// 3. Start Stdio Service
	// Stdio mode allows Claude Desktop to communicate via stdin/stdout
	if err := s.ServeStdio(); err != nil {
		panic(err)
	}
}
```

## 3. Build

```bash
go build -o weather-server main.go
```

## 4. Integrate with Claude Desktop

To let Claude Desktop use your MCP Server, you need to modify the configuration file.

### macOS

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "go-weather": {
      "command": "/absolute/path/to/your/weather-server"
    }
  }
}
```

Save the file and restart Claude Desktop. You should see a 🔌 icon and be able to ask Claude "What is the weather in Beijing?".

## 5. Remote Deployment (WebSocket)

If you want to deploy the service on a remote server, simply change the startup code:

```go
// Start WebSocket Service
if err := s.ServeWebSocket(":8080"); err != nil {
    panic(err)
}
```

Clients can then connect via `ws://your-server-ip:8080/mcp`.
