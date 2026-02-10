# 快速开始指南

本文档将指导你从零开始创建一个 GoT0MCP 服务，并将其集成到 Claude Desktop 或其他 MCP 客户端中。

## 前置要求

- **Go**: 1.21 或更高版本
- **Git**: 用于下载代码库

## 1. 安装

在你的 Go 项目中执行：

```bash
go get github.com/Wh1teJ0ker/GoT0MCP
```

## 2. 编写第一个 MCP Server

创建一个 `main.go` 文件：

```go
package main

import (
	"context"
	"fmt"
	"github.com/Wh1teJ0ker/GoT0MCP/pkg/mcp"
)

// WeatherArgs 定义查询天气的参数
type WeatherArgs struct {
	City string `json:"city" jsonschema:"description=要查询的城市名称"`
}

func main() {
	// 1. 初始化 Server
	s := mcp.NewServer(mcp.ServerOptions{
		Name: "WeatherServer",
		Version: "0.1.0",
	})

	// 2. 注册工具
	s.AddTool("get_weather", "获取指定城市的天气", func(ctx context.Context, args WeatherArgs) (string, error) {
		// 这里可以是实际的 API 调用
		return fmt.Sprintf("%s 的天气是晴天，25°C", args.City), nil
	})

	// 3. 启动 Stdio 服务
	// Stdio 模式允许 Claude Desktop 通过标准输入输出与服务通信
	if err := s.ServeStdio(); err != nil {
		panic(err)
	}
}
```

## 3. 编译

```bash
go build -o weather-server main.go
```

## 4. 集成到 Claude Desktop

要让 Claude Desktop 使用你的 MCP Server，你需要修改配置文件。

### macOS

编辑 `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "go-weather": {
      "command": "/绝对路径/指向/你的/weather-server"
    }
  }
}
```

保存文件并重启 Claude Desktop。你应该能看到一个 🔌 图标，并且可以询问 Claude "北京的天气怎么样？"。

## 5. 远程部署 (WebSocket)

如果你想将服务部署在远程服务器上，只需更改启动代码：

```go
// 启动 WebSocket 服务
if err := s.ServeWebSocket(":8080"); err != nil {
    panic(err)
}
```

然后客户端可以通过 `ws://your-server-ip:8080/mcp` 连接。
