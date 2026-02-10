# GoT0MCP

[![Go Reference](https://pkg.go.dev/badge/github.com/Wh1teJ0ker/GoT0MCP.svg)](https://pkg.go.dev/github.com/Wh1teJ0ker/GoT0MCP)
[![Go Report Card](https://goreportcard.com/badge/github.com/Wh1teJ0ker/GoT0MCP)](https://goreportcard.com/report/github.com/Wh1teJ0ker/GoT0MCP)
[![Build Status](https://github.com/Wh1teJ0ker/GoT0MCP/actions/workflows/benchmark.yml/badge.svg)](https://github.com/Wh1teJ0ker/GoT0MCP/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

[English](README_EN.md) | [中文](README.md)

GoT0MCP 是一个专为 Go 语言设计的高性能 Model Context Protocol (MCP) 服务端库。它借鉴了 Python FastMCP 的设计理念，旨在提供极致简化的开发体验，同时充分释放 Go 语言在高并发与强类型系统方面的优势。

## 核心特性

*   **极速开发体验**：通过反射机制实现 Go 函数到 MCP 工具的自动注册，无需手动编写繁琐的 JSON Schema，让开发者专注于业务逻辑。
*   **高性能架构**：底层集成 `goccy/go-json` 高性能 JSON 引擎，配合 Server/Session 分离的并发架构，在基准测试中可达 40,000+ QPS。
*   **生产级健壮性**：内置超时熔断、Panic 自动捕获与恢复机制，配合 Prometheus 监控埋点，确保生产环境的稳定性与可观测性。
*   **双协议支持**：开箱即用支持 Stdio（本地 CLI/IDE 集成）与 WebSocket（分布式/远程服务）两种传输协议。

## 安装

```bash
go get github.com/Wh1teJ0ker/GoT0MCP
```

## 快速开始

以下示例展示了如何创建一个简单的加法工具并启动服务。

```go
package main

import (
	"context"
	"github.com/Wh1teJ0ker/GoT0MCP/pkg/mcp"
)

// AddArgs 定义参数结构体
// 支持通过 jsonschema 标签自定义描述
type AddArgs struct {
	A int `json:"a" jsonschema:"description=加数 A"`
	B int `json:"b" jsonschema:"description=加数 B"`
}

func main() {
	// 1. 创建服务器实例
	s := mcp.NewServer()

	// 2. 注册工具
	// 自动解析函数签名生成 MCP Tool Schema
	s.AddTool("add", "计算两个整数之和", func(ctx context.Context, args AddArgs) (int, error) {
		return args.A + args.B, nil
	})

	// 3. 启动服务 (默认使用 Stdio 传输)
	if err := s.ServeStdio(); err != nil {
		panic(err)
	}
}
```

## 性能表现

在 MacBook Pro (M1 Pro) 环境下的基准测试结果：

| 测试场景 | Payload 大小 | 并发数 | QPS | 平均延迟 |
| :--- | :--- | :--- | :--- | :--- |
| 基准吞吐量 | Small | 50 | **41,953** | 1.06ms |
| 数据传输 | 1KB | 20 | **32,076** | 0.52ms |
| 数据传输 | 10KB | 20 | **15,482** | 1.15ms |
| 高负载 | 100KB | 10 | **3,241** | 2.42ms |

> 测试脚本位于 `scripts/benchmark.sh`

## 工具可视化

GoT0MCP 提供了直观的工具注册与管理能力，支持嵌套结构体参数解析。

```text
Server
├── Tools
│   ├── add (a: int, b: int)
│   ├── concat (str1: string, str2: string)
│   └── process_user (user: UserArgs)
│       └── UserArgs
│           ├── Name (string)
│           ├── Age (int)
│           └── Address (struct)
│               ├── Street (string)
│               └── City (string)
└── Transports
    ├── Stdio (Standard I/O)
    └── WebSocket (JSON-RPC 2.0)
```

## 文档

*   [快速入门](docs/zh-CN/getting-started.md)
*   [架构设计](docs/zh-CN/architecture.md)

## 许可证

本项目采用 MIT 许可证。
