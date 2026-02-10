package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Wh1teJ0ker/GoT0MCP/pkg/mcp"
)

// CalcArgs 定义加法工具的参数
type CalcArgs struct {
	A int `json:"a" jsonschema:"description=第一个数字"`
	B int `json:"b" jsonschema:"description=第二个数字"`
}

// ConcatArgs 定义字符串连接工具的参数
type ConcatArgs struct {
	Str1 string `json:"str1" jsonschema:"description=第一个字符串"`
	Str2 string `json:"str2" jsonschema:"description=第二个字符串"`
}

type Address struct {
	Street string `json:"street" jsonschema:"description=街道名称"`
	City   string `json:"city" jsonschema:"description=城市名称"`
}

type UserArgs struct {
	Name    string  `json:"name" jsonschema:"description=用户姓名"`
	Age     int     `json:"age" jsonschema:"description=用户年龄"`
	Address Address `json:"address" jsonschema:"description=详细地址"`
}

type EchoArgs struct {
	Payload string `json:"payload" jsonschema:"description=Payload content"`
}

type DelayArgs struct {
	Seconds int `json:"seconds" jsonschema:"description=Delay duration in seconds"`
}

func main() {
	// 注册 Prometheus Metrics Handler
	http.Handle("/metrics", promhttp.Handler())

	// 配置日志
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// 创建 MCP 服务器实例，带有自定义配置
	s := mcp.NewServer(mcp.ServerOptions{
		Name:        "ExampleServer",
		Version:     "1.0.0",
		Logger:      logger,
		ToolTimeout: 10 * time.Second, // 设置工具超时为 10 秒
	})

	// 注册加法工具
	// 使用反射自动生成 Schema 并处理调用
	err := s.AddTool("add", "计算两个数字之和", func(ctx context.Context, args CalcArgs) (int, error) {
		// 模拟耗时操作，测试超时
		// time.Sleep(2 * time.Second)
		fmt.Printf("调用 add 工具: a=%d, b=%d\n", args.A, args.B)
		return args.A + args.B, nil
	})
	if err != nil {
		panic(err)
	}

	// 注册字符串连接工具
	err = s.AddTool("concat", "连接两个字符串", func(ctx context.Context, args ConcatArgs) (string, error) {
		fmt.Printf("调用 concat 工具: str1=%s, str2=%s\n", args.Str1, args.Str2)
		return args.Str1 + args.Str2, nil
	})
	if err != nil {
		panic(err)
	}

	// 注册复杂对象工具
	err = s.AddTool("process_user", "处理用户信息（测试嵌套对象）", func(ctx context.Context, args UserArgs) (string, error) {
		return fmt.Sprintf("User: %s, Age: %d, Address: %s, %s", args.Name, args.Age, args.Address.Street, args.Address.City), nil
	})
	if err != nil {
		panic(err)
	}

	// 注册 Echo 工具 (Payload 测试)
	err = s.AddTool("echo", "Echo payload for bandwidth testing", func(ctx context.Context, args EchoArgs) (string, error) {
		// 返回 Payload 的长度信息，避免日志过大，但实际返回原内容
		slog.Debug("Echo tool called", "size", len(args.Payload))
		return args.Payload, nil
	})
	if err != nil {
		panic(err)
	}

	// 注册 Delay 工具 (故障注入)
	err = s.AddTool("delay", "Simulate processing delay", func(ctx context.Context, args DelayArgs) (string, error) {
		time.Sleep(time.Duration(args.Seconds) * time.Second)
		return fmt.Sprintf("Delayed for %d seconds", args.Seconds), nil
	})
	if err != nil {
		panic(err)
	}

	// 根据命令行参数选择启动模式
	// 默认为 Stdio 模式
	if len(os.Args) > 1 && os.Args[1] == "websocket" {
		addr := ":8082"
		if len(os.Args) > 2 {
			addr = os.Args[2]
		}
		// 启动 WebSocket 服务
		if err := s.ServeWebSocket(addr); err != nil {
			panic(err)
		}
	} else {
		// 启动 Stdio 服务 (默认)
		if err := s.ServeStdio(); err != nil {
			panic(err)
		}
	}
}
