# Architecture Overview

GoT0MCP is designed for high performance, extensibility, and type safety. This document details its core architectural components.

## Architecture Layers

GoT0MCP adopts a layered architecture:

1.  **Transport Layer**: Handles low-level JSON-RPC message sending and receiving.
2.  **Session Layer**: Manages the state, request lifecycle, and concurrency control for a single connection.
3.  **Server Layer**: Global singleton managing the tool registry, configuration, and session factory.
4.  **Reflection Layer**: Handles the automatic mapping between Go types and MCP Schemas.

## Core Components

### 1. Server (pkg/mcp/server.go)

`Server` is the entry point. It acts as a global state container responsible for:

- **Tool Registration**: Maintaining the `tools` list and `toolHandlers` map.
- **Configuration Management**: Storing `ServerOptions` (e.g., Logger, Timeout).
- **Session Factory**: Creating new session instances via `NewSession`.

```go
type Server struct {
    tools        []abi.ToolSpec
    toolHandlers map[string]ToolHandler
    options      ServerOptions
    logger       *slog.Logger
}
```

### 2. Session (pkg/mcp/server.go)

`Session` represents an active client connection. For WebSocket, each connection corresponds to a Session; for Stdio, there is typically only one Session.

- **Request Routing**: Receives data from the Transport layer and parses JSON-RPC messages.
- **Concurrency Handling**: Launches a new Goroutine for each request.
- **Context Management**: Creates a Context with timeout control for each request.
- **Panic Recovery**: Captures Panics in user code to prevent service crashes.

```go
type Session struct {
    server       *Server
    transport    transport.Transport
    pendingReqs  map[string]chan *transport.JSONRPCMessage
    // ...
}
```

### 3. Reflection (pkg/mcp/reflection.go)

This is the core of "Fast". The `AddTool` method leverages Go's reflection mechanism:

1.  **Type Checking**: Verifies if the user-provided function signature matches `func(context.Context, Args) (Result, error)`.
2.  **Schema Generation**: Recursively parses the `Args` struct, reading `json` and `jsonschema` tags to generate an MCP-compliant JSON Schema.
3.  **Argument Binding**: Deserializes JSON request parameters into Go struct instances at runtime.

### 4. Transport (pkg/mcp/transport)

Defines the `Transport` interface to decouple protocol implementations:

```go
type Transport interface {
    Read() (*JSONRPCMessage, error)
    Write(*JSONRPCMessage) error
    Close() error
}
```

Currently provides two implementations:
- **Stdio**: Uses `os.Stdin` and `os.Stdout`, suitable for CLI tools and local integration.
- **WebSocket**: Uses `github.com/gorilla/websocket`, suitable for remote services.

## Concurrency Model

GoT0MCP fully leverages Go's Goroutines:

1.  **Transport Loop**: Each Session starts a `Read` loop to continuously read messages.
2.  **Request Handling**: Upon receiving a Request, it immediately launches `go session.handleRequest(msg)`.
3.  **Tool Execution**: User-written tool functions run in independent Goroutines, never blocking each other.

This design ensures that slow tool calls do not block the processing of other requests, supporting high throughput.
