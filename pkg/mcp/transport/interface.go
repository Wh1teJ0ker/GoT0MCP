package transport

import "encoding/json"

// Transport 定义了 MCP 通信的接口
type Transport interface {
	Read() (*JSONRPCMessage, error)
	Write(*JSONRPCMessage) error
	Close() error
}

// JSONRPCMessage 代表一个基本的 JSON-RPC 2.0 消息
type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
