#!/bin/bash

# Configuration
RESULTS_DIR="test_results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_FILE="$RESULTS_DIR/report_${TIMESTAMP}.md"
SERVER_PORT=8082
METRICS_PORT=8082 # Same port in current impl

mkdir -p "$RESULTS_DIR"

# Build
echo "🏗️  Building binaries..."
go build -o bin/server cmd/server/main.go
go build -o bin/load_tester cmd/load_tester/main.go

# Start Server
echo "🚀 Starting MCP Server..."
./bin/server websocket :$SERVER_PORT > "$RESULTS_DIR/server_${TIMESTAMP}.log" 2>&1 &
SERVER_PID=$!

# Wait for server
sleep 3
if ! ps -p $SERVER_PID > /dev/null; then
    echo "❌ Server failed to start. Check logs."
    exit 1
fi

echo "✅ Server running (PID: $SERVER_PID)"

# Function to run test
run_test() {
    NAME=$1
    ARGS=$2
    echo "🧪 Running Test: $NAME"
    echo "   Args: $ARGS"
    
    echo "## Test Case: $NAME" >> "$REPORT_FILE"
    echo "\`\`\`" >> "$REPORT_FILE"
    ./bin/load_tester $ARGS | tee -a "$REPORT_FILE"
    echo "\`\`\`" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # Capture Metrics (Snapshot)
    echo "### Metrics Snapshot" >> "$REPORT_FILE"
    echo "\`\`\`" >> "$REPORT_FILE"
    curl -s http://localhost:$METRICS_PORT/metrics | grep "mcp_" >> "$REPORT_FILE"
    echo "\`\`\`" >> "$REPORT_FILE"
    echo "---" >> "$REPORT_FILE"
    
    sleep 2
}

# Initialize Report
cat <<EOF > "$REPORT_FILE"
# 📊 MCP High-Performance Burst Test Report
**Date:** $(date)
**Environment:** MacOS (Dev)
**Server Version:** 1.0.0 (GoT0MCP)

## 1. Test Environment Configuration
- **CPU/Mem:** (Auto-detected via Go runtime)
- **Transport:** WebSocket
- **JSON Engine:** goccy/go-json

## 2. Benchmark Results
EOF

# 1. Baseline Test (Small Payload, High Concurrency)
# Simulating high concurrency (50 concurrent connections, 5000 requests)
run_test "Baseline - Add Tool (Small Payload)" "-c 50 -n 5000 -tool add -qps 0"

# 2. Payload Size Comparison
run_test "Payload 1KB" "-c 20 -n 2000 -tool echo -size 1024"
run_test "Payload 10KB" "-c 20 -n 2000 -tool echo -size 10240"
run_test "Payload 100KB" "-c 10 -n 500 -tool echo -size 102400"

# 3. Ramp-up Test
run_test "Ramp-up Stress Test" "-c 100 -n 5000 -tool add -ramp 5s"

# 4. Fault Injection (Delay)
# 100 requests, expecting ~10s total time due to parallel execution of 1s delays
run_test "Fault Injection - 1s Delay" "-c 50 -n 100 -tool delay"

# Cleanup
echo "🛑 Stopping Server..."
kill $SERVER_PID

echo "📝 Report generated at: $REPORT_FILE"
