package mcp

import (
	"net/http"

	"github.com/Wh1teJ0ker/GoT0MCP/pkg/mcp/transport"
	"github.com/gorilla/websocket"
)

// ServeStdio 使用 Stdio 传输启动服务器
// 这将创建一个单一的会话并阻塞直到结束
func (s *Server) ServeStdio() error {
	s.logger.Info("启动 Stdio 服务")
	tr := transport.NewStdio()
	session := s.NewSession(tr)
	return session.Start()
}

// ServeWebSocket 在指定地址使用 WebSocket 传输启动服务器
// 这将启动一个 HTTP 服务器，为每个 WebSocket 连接创建一个新的会话
func (s *Server) ServeWebSocket(addr string) error {
	http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Error("WebSocket 升级失败", "error", err)
			return
		}
		tr := transport.NewWebSocket(conn)

		// 为新连接创建一个会话
		session := s.NewSession(tr)

		// 异步启动会话处理
		go func() {
			if err := session.Start(); err != nil {
				// 连接关闭或出错
				s.logger.Info("WebSocket 会话结束", "error", err)
			}
		}()
	})

	s.logger.Info("WebSocket 服务启动", "addr", addr)
	return http.ListenAndServe(addr, nil)
}
