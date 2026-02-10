package mcp

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// mcpActiveSessions 当前活跃会话数
	mcpActiveSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mcp_active_sessions",
		Help: "The number of currently active MCP sessions",
	})

	// mcpRequestsTotal 总请求数
	mcpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mcp_requests_total",
		Help: "The total number of requests processed",
	}, []string{"method", "status"})

	// mcpRequestDuration 请求处理延迟
	mcpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mcp_request_duration_seconds",
		Help:    "The duration of request processing in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method"})

	// mcpPendingRequests 当前挂起的请求数（队列深度）
	mcpPendingRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mcp_pending_requests",
		Help: "The number of currently pending requests",
	})

	// mcpMessageSize 消息大小分布
	mcpMessageSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mcp_message_size_bytes",
		Help:    "The size of messages in bytes",
		Buckets: []float64{100, 1024, 10240, 102400, 1048576, 4194304}, // 100B, 1KB, 10KB, 100KB, 1MB, 4MB
	}, []string{"direction"}) // "in" or "out"
)
