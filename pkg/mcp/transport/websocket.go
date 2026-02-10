package transport

import (
	"sync"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
)

// WebSocket 处理基于 WebSocket 的 JSON-RPC
type WebSocket struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func NewWebSocket(conn *websocket.Conn) *WebSocket {
	return &WebSocket{
		conn: conn,
	}
}

func (w *WebSocket) Read() (*JSONRPCMessage, error) {
	_, p, err := w.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	var msg JSONRPCMessage
	if err := json.Unmarshal(p, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (w *WebSocket) Write(msg *JSONRPCMessage) error {
	p, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteMessage(websocket.TextMessage, p)
}

func (w *WebSocket) Close() error {
	return w.conn.Close()
}
