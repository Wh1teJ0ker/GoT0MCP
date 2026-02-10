package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"

	"github.com/Wh1teJ0ker/GoT0MCP/pkg/abi"
	"github.com/Wh1teJ0ker/GoT0MCP/pkg/mcp/transport"
)

// ToolHandler 定义工具调用的函数签名
type ToolHandler func(ctx context.Context, args json.RawMessage) (map[string]any, error)

// ServerOptions 定义服务器配置选项
type ServerOptions struct {
	// Name 服务器名称
	Name string
	// Version 服务器版本
	Version string
	// Logger 日志记录器，如果为 nil 则使用默认 slog.Logger
	Logger *slog.Logger
	// ToolTimeout 工具执行超时时间，默认为 30 秒
	ToolTimeout time.Duration
}

// Server 代表 MCP 服务器，负责管理工具注册
type Server struct {
	// 注册的工具
	tools        []abi.ToolSpec
	toolHandlers map[string]ToolHandler

	// 配置
	name        string
	version     string
	logger      *slog.Logger
	toolTimeout time.Duration
}

// NewServer 创建一个新的 MCP 服务器实例
// 可以传入可选的 ServerOptions
func NewServer(opts ...ServerOptions) *Server {
	options := ServerOptions{
		Name:        "GoT0MCP",
		Version:     "0.1.0",
		Logger:      slog.Default(),
		ToolTimeout: 30 * time.Second,
	}
	if len(opts) > 0 {
		if opts[0].Name != "" {
			options.Name = opts[0].Name
		}
		if opts[0].Version != "" {
			options.Version = opts[0].Version
		}
		if opts[0].Logger != nil {
			options.Logger = opts[0].Logger
		}
		if opts[0].ToolTimeout > 0 {
			options.ToolTimeout = opts[0].ToolTimeout
		}
	}

	return &Server{
		toolHandlers: make(map[string]ToolHandler),
		name:         options.Name,
		version:      options.Version,
		logger:       options.Logger,
		toolTimeout:  options.ToolTimeout,
	}
}

// RegisterTool 注册一个工具到服务器
func (s *Server) RegisterTool(spec abi.ToolSpec, handler ToolHandler) {
	s.logger.Info("注册工具", "name", spec.Name, "description", spec.Description)
	s.tools = append(s.tools, spec)
	s.toolHandlers[spec.Name] = handler
}

// Session 代表一个活跃的 MCP 连接会话
type Session struct {
	server      *Server
	transport   transport.Transport
	pendingReqs map[string]chan *transport.JSONRPCMessage
	clientTools []abi.ToolSpec
	mu          sync.Mutex
	requestID   atomic.Int64
}

// NewSession 创建一个新的会话
func (s *Server) NewSession(tr transport.Transport) *Session {
	mcpActiveSessions.Inc()
	return &Session{
		server:      s,
		transport:   tr,
		pendingReqs: make(map[string]chan *transport.JSONRPCMessage),
	}
}

// Start 开始处理会话的消息循环
func (sess *Session) Start() error {
	sess.server.logger.Info("会话开始")
	defer func() {
		mcpActiveSessions.Dec()
		sess.server.logger.Info("会话结束")
	}()

	for {
		msg, err := sess.transport.Read()
		if err != nil {
			sess.server.logger.Error("读取消息失败", "error", err)
			return err
		}

		if msg.Method != "" {
			// 请求或通知
			// 异步处理以支持并发请求
			go sess.handleRequest(msg)
		} else if msg.ID != nil {
			// 响应
			sess.handleResponse(msg)
		}
	}
}

func (sess *Session) handleRequest(msg *transport.JSONRPCMessage) {
	start := time.Now()
	method := msg.Method
	// 如果是空方法，可能是响应，不应进入这里，但以防万一
	if method == "" {
		method = "unknown"
	}

	defer func() {
		duration := time.Since(start).Seconds()
		mcpRequestDuration.WithLabelValues(method).Observe(duration)
	}()

	sess.server.logger.Debug("收到请求", "method", msg.Method, "id", string(msg.ID))

	// 捕获 panic 以防止单个请求崩溃导致整个会话中断
	defer func() {
		if r := recover(); r != nil {
			mcpRequestsTotal.WithLabelValues(method, "error").Inc()
			sess.server.logger.Error("处理请求时发生 panic", "method", msg.Method, "panic", r, "stack", string(debug.Stack()))
			if msg.ID != nil {
				sess.reply(msg.ID, map[string]any{
					"isError": true,
					"content": []map[string]string{
						{
							"type": "text",
							"text": fmt.Sprintf("Internal Server Error: %v", r),
						},
					},
				})
			}
		}
	}()

	switch msg.Method {
	case "initialize":
		sess.reply(msg.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{
					"listChanged": true,
				},
			},
			"serverInfo": map[string]any{
				"name":    sess.server.name,
				"version": sess.server.version,
			},
		})
	case "notifications/tools/list_changed":
		// 客户端工具列表变更，重新获取
		sess.fetchClientTools()
	case "tools/list":
		sess.reply(msg.ID, map[string]any{
			"tools": sess.server.tools,
		})
	case "tools/call":
		sess.handleToolCall(msg)
	default:
		// 忽略或返回错误
		sess.server.logger.Warn("收到未知方法", "method", msg.Method)
	}
}

func (sess *Session) handleResponse(msg *transport.JSONRPCMessage) {
	sess.mu.Lock()
	defer sess.mu.Unlock()

	idStr := string(msg.ID)
	if ch, ok := sess.pendingReqs[idStr]; ok {
		ch <- msg
		delete(sess.pendingReqs, idStr)
	}
}

func (sess *Session) reply(id json.RawMessage, result any) {
	resBytes, _ := json.Marshal(result)
	err := sess.transport.Write(&transport.JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  resBytes,
	})
	if err != nil {
		sess.server.logger.Error("发送响应失败", "error", err)
	}
}

func (sess *Session) sendRequest(method string, params any) (*transport.JSONRPCMessage, error) {
	id := fmt.Sprintf("%d", sess.requestID.Add(1))
	idRaw := json.RawMessage(`"` + id + `"`)

	ch := make(chan *transport.JSONRPCMessage, 1)
	sess.mu.Lock()
	sess.pendingReqs[`"`+id+`"`] = ch // JSON raw message 包含引号
	mcpPendingRequests.Inc()
	sess.mu.Unlock()

	paramsBytes, _ := json.Marshal(params)
	err := sess.transport.Write(&transport.JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      idRaw,
		Method:  method,
		Params:  paramsBytes,
	})
	if err != nil {
		sess.mu.Lock()
		delete(sess.pendingReqs, `"`+id+`"`)
		mcpPendingRequests.Dec()
		sess.mu.Unlock()
		return nil, err
	}

	// 等待响应，增加超时控制
	select {
	case resp := <-ch:
		sess.mu.Lock()
		mcpPendingRequests.Dec()
		sess.mu.Unlock()
		return resp, nil
	case <-time.After(sess.server.toolTimeout): // 复用工具超时配置作为请求超时
		sess.mu.Lock()
		delete(sess.pendingReqs, `"`+id+`"`)
		mcpPendingRequests.Dec()
		sess.mu.Unlock()
		return nil, fmt.Errorf("request timeout")
	}
}

func (sess *Session) fetchClientTools() {
	// 向客户端请求 tools/list
	resp, err := sess.sendRequest("tools/list", nil)
	if err != nil {
		sess.server.logger.Error("获取客户端工具失败", "error", err)
		return
	}

	var result struct {
		Tools []abi.ToolSpec `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err == nil {
		sess.clientTools = result.Tools
		sess.server.logger.Info("更新客户端工具列表", "count", len(sess.clientTools))
	}
}

func (sess *Session) handleToolCall(msg *transport.JSONRPCMessage) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		sess.server.logger.Error("解析工具调用参数失败", "error", err)
		sess.reply(msg.ID, map[string]any{
			"isError": true,
			"content": []map[string]string{
				{
					"type": "text",
					"text": fmt.Sprintf("Invalid params: %v", err),
				},
			},
		})
		return
	}

	sess.server.logger.Info("执行工具调用", "tool", params.Name)

	if handler, ok := sess.server.toolHandlers[params.Name]; ok {
		// 创建带超时的 Context
		ctx, cancel := context.WithTimeout(context.Background(), sess.server.toolTimeout)
		defer cancel()

		result, err := handler(ctx, params.Arguments)
		if err != nil {
			sess.server.logger.Error("工具执行出错", "tool", params.Name, "error", err)
			sess.reply(msg.ID, map[string]any{
				"isError": true,
				"content": []map[string]string{
					{
						"type": "text",
						"text": fmt.Sprintf("Tool execution error: %v", err),
					},
				},
			})
		} else {
			sess.server.logger.Info("工具执行成功", "tool", params.Name)
			mcpRequestsTotal.WithLabelValues("tools/call/"+params.Name, "success").Inc()
			sess.reply(msg.ID, result)
		}
	} else {
		sess.server.logger.Warn("工具未找到", "tool", params.Name)
		mcpRequestsTotal.WithLabelValues("tools/call", "not_found").Inc()
		sess.reply(msg.ID, map[string]any{
			"isError": true,
			"content": []map[string]string{
				{
					"type": "text",
					"text": "Tool not found: " + params.Name,
				},
			},
		})
	}
}
