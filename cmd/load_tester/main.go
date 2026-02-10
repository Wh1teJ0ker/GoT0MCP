package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"
	"github.com/gorilla/websocket"
)

// Config holds the load test configuration
type Config struct {
	Concurrency int
	TotalReqs   int
	QPS         int
	Tool        string
	PayloadSize int
	RampUp      time.Duration
	URL         string
}

// Stats holds the test results
type Stats struct {
	TotalRequests int64
	Success       int64
	Failures      int64
	Latencies     []time.Duration
	StartTime     time.Time
	EndTime       time.Time
	mu            sync.Mutex
}

func (s *Stats) AddLatency(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Latencies = append(s.Latencies, d)
}

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
}

func main() {
	c := flag.Int("c", 10, "Concurrency level")
	n := flag.Int("n", 1000, "Total requests")
	qps := flag.Int("qps", 0, "Target QPS (0 for unlimited)")
	tool := flag.String("tool", "add", "Tool to call: add, echo, delay")
	size := flag.Int("size", 1024, "Payload size for echo tool (bytes)")
	ramp := flag.Duration("ramp", 0, "Ramp up duration")
	host := flag.String("host", "ws://localhost:8082/mcp", "MCP Server URL")

	flag.Parse()

	config := Config{
		Concurrency: *c,
		TotalReqs:   *n,
		QPS:         *qps,
		Tool:        *tool,
		PayloadSize: *size,
		RampUp:      *ramp,
		URL:         *host,
	}

	runLoadTest(config)
}

func runLoadTest(cfg Config) {
	log.Printf("Starting load test: Concurrency=%d, Total=%d, Tool=%s, URL=%s", cfg.Concurrency, cfg.TotalReqs, cfg.Tool, cfg.URL)

	stats := &Stats{
		TotalRequests: int64(cfg.TotalReqs),
		StartTime:     time.Now(),
		Latencies:     make([]time.Duration, 0, cfg.TotalReqs),
	}

	var wg sync.WaitGroup
	requestsPerWorker := cfg.TotalReqs / cfg.Concurrency

	// Payload generation
	payload := ""
	if cfg.Tool == "echo" {
		payload = strings.Repeat("a", cfg.PayloadSize)
	}

	// Rate limiter
	var limiter <-chan time.Time
	if cfg.QPS > 0 {
		ticker := time.NewTicker(time.Second / time.Duration(cfg.QPS))
		defer ticker.Stop()
		limiter = ticker.C
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		<-sigChan
		log.Println("Stopping load test...")
		cancel()
	}()

	// Circuit breaker counters
	var errorCount int64
	var totalCount int64

	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Ramp up delay
			if cfg.RampUp > 0 {
				delay := time.Duration(float64(cfg.RampUp) * float64(workerID) / float64(cfg.Concurrency))
				time.Sleep(delay)
			}

			// Connect
			conn, _, err := websocket.DefaultDialer.Dial(cfg.URL, nil)
			if err != nil {
				log.Printf("Worker %d failed to connect: %v", workerID, err)
				atomic.AddInt64(&stats.Failures, 1)
				return
			}
			defer conn.Close()

			// Read loop (discard responses for now, or match IDs if strictly needed,
			// but for load testing generic throughput, we assume request-response pairs)
			// Ideally we need a proper client that matches IDs.
			// For simplicity, we'll read one message for every write in the loop.

			// Initial Handshake
			// Send initialize
			initReq := JSONRPCMessage{
				JSONRPC: "2.0",
				ID:      json.RawMessage(`1`),
				Method:  "initialize",
			}
			writeJSON(conn, initReq)
			_, _, _ = conn.ReadMessage() // Read response

			for j := 0; j < requestsPerWorker; j++ {
				if ctx.Err() != nil {
					return
				}

				// Rate limiting
				if limiter != nil {
					<-limiter
				}

				// Check circuit breaker (e.g. > 50% error rate after 100 requests)
				t := atomic.LoadInt64(&totalCount)
				e := atomic.LoadInt64(&errorCount)
				if t > 100 && float64(e)/float64(t) > 0.5 {
					log.Println("Circuit breaker tripped! Stopping test.")
					cancel()
					return
				}

				reqID := fmt.Sprintf("%d-%d", workerID, j)
				req := buildRequest(reqID, cfg.Tool, payload)

				start := time.Now()
				if err := writeJSON(conn, req); err != nil {
					atomic.AddInt64(&stats.Failures, 1)
					atomic.AddInt64(&errorCount, 1)
					atomic.AddInt64(&totalCount, 1)
					continue
				}

				// Read response
				_, msg, err := conn.ReadMessage()
				latency := time.Since(start)
				atomic.AddInt64(&totalCount, 1)

				if err != nil {
					atomic.AddInt64(&stats.Failures, 1)
					atomic.AddInt64(&errorCount, 1)
				} else {
					// Check for JSON-RPC error
					var resp JSONRPCMessage
					if err := json.Unmarshal(msg, &resp); err == nil && resp.Error != nil {
						atomic.AddInt64(&stats.Failures, 1)
						atomic.AddInt64(&errorCount, 1)
					} else {
						atomic.AddInt64(&stats.Success, 1)
						stats.AddLatency(latency)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	stats.EndTime = time.Now()
	printStats(stats)
}

func writeJSON(conn *websocket.Conn, v interface{}) error {
	w, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}
	err = json.NewEncoder(w).Encode(v)
	if err != nil {
		return err
	}
	return w.Close()
}

func buildRequest(id, tool, payload string) JSONRPCMessage {
	idRaw := json.RawMessage(`"` + id + `"`)
	var params json.RawMessage

	switch tool {
	case "add":
		params = json.RawMessage(`{"name":"add","arguments":{"a":1,"b":2}}`)
	case "echo":
		// Construct JSON manually to avoid struct overhead in hot path
		p := fmt.Sprintf(`{"name":"echo","arguments":{"payload":"%s"}}`, payload)
		params = json.RawMessage(p)
	case "delay":
		params = json.RawMessage(`{"name":"delay","arguments":{"seconds":1}}`)
	default:
		params = json.RawMessage(`{}`)
	}

	return JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      idRaw,
		Method:  "tools/call",
		Params:  params,
	}
}

func printStats(s *Stats) {
	duration := s.EndTime.Sub(s.StartTime).Seconds()
	qps := float64(s.Success) / duration

	var totalLatency time.Duration
	var maxLatency time.Duration
	for _, l := range s.Latencies {
		totalLatency += l
		if l > maxLatency {
			maxLatency = l
		}
	}
	avgLatency := time.Duration(0)
	if len(s.Latencies) > 0 {
		avgLatency = totalLatency / time.Duration(len(s.Latencies))
	}

	fmt.Printf("\n=== Load Test Results ===\n")
	fmt.Printf("Duration: %.2fs\n", duration)
	fmt.Printf("Total Requests: %d\n", s.TotalRequests) // This might be slightly off due to partial updates in struct, but counters are atomic
	fmt.Printf("Success: %d\n", s.Success)
	fmt.Printf("Failures: %d\n", s.Failures)
	fmt.Printf("QPS: %.2f\n", qps)
	fmt.Printf("Avg Latency: %v\n", avgLatency)
	fmt.Printf("Max Latency: %v\n", maxLatency)
}
