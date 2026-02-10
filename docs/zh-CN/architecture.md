# 架构详解

GoT0MCP 的设计目标是高性能、易扩展和类型安全。本文档详细介绍了其核心架构组件。

## 架构概览

GoT0MCP 采用了分层架构，主要分为以下几个层次：

1.  **Transport Layer (传输层)**: 负责底层的 JSON-RPC 消息收发。
2.  **Session Layer (会话层)**: 管理单个连接的状态、请求生命周期和并发控制。
3.  **Server Layer (服务层)**: 全局单例，管理工具注册表、配置和会话工厂。
4.  **Reflection Layer (反射层)**: 负责 Go 类型与 MCP Schema 之间的自动映射。

## 核心组件

### 1. Server (pkg/mcp/server.go)

`Server` 是库的入口点。它是一个全局状态容器，负责：

- **工具注册**: 维护 `tools` 列表和 `toolHandlers` 映射。
- **配置管理**: 存储 `ServerOptions`（如 Logger, Timeout）。
- **会话工厂**: 通过 `NewSession` 创建新的会话实例。

```go
type Server struct {
    tools        []abi.ToolSpec
    toolHandlers map[string]ToolHandler
    options      ServerOptions
    logger       *slog.Logger
}
```

### 2. Session (pkg/mcp/server.go)

`Session` 代表一个活跃的客户端连接。对于 WebSocket，每个连接对应一个 Session；对于 Stdio，通常只有一个 Session。

- **请求路由**: 接收 Transport 层的数据，解析 JSON-RPC 消息。
- **并发处理**: 为每个请求启动一个新的 Goroutine。
- **上下文管理**: 为每个请求创建带有超时控制的 Context。
- **Panic 恢复**: 捕获用户代码中的 Panic，防止服务崩溃。

```go
type Session struct {
    server       *Server
    transport    transport.Transport
    pendingReqs  map[string]chan *transport.JSONRPCMessage
    // ...
}
```

### 3. Reflection (pkg/mcp/reflection.go)

这是 "Fast" 的核心。`AddTool` 方法利用 Go 的反射机制：

1.  **类型检查**: 验证用户传入的函数签名是否符合 `func(context.Context, Args) (Result, error)`。
2.  **Schema 生成**: 递归解析 `Args` 结构体，读取 `json` 和 `jsonschema` 标签，生成符合 MCP 规范的 JSON Schema。
3.  **参数绑定**: 在运行时，将 JSON 请求参数反序列化为 Go 结构体实例。

### 4. Transport (pkg/mcp/transport)

定义了 `Transport` 接口，解耦了协议实现：

```go
type Transport interface {
    Read() (*JSONRPCMessage, error)
    Write(*JSONRPCMessage) error
    Close() error
}
```

目前提供两种实现：
- **Stdio**: 使用 `os.Stdin` 和 `os.Stdout`，适合 CLI 工具和本地集成。
- **WebSocket**: 使用 `github.com/gorilla/websocket`，适合远程服务。

## 并发模型

GoT0MCP 充分利用了 Go 的 Goroutine：

1.  **Transport Loop**: 每个 Session 启动一个 `Read` 循环，持续读取消息。
2.  **Request Handling**: 收到 Request 后，立即启动 `go session.handleRequest(msg)`。
3.  **Tool Execution**: 用户编写的工具函数在独立的 Goroutine 中运行，互不阻塞。

这种设计确保了慢速的工具调用不会阻塞其他请求的处理，从而支持高吞吐量。
