package transport

import (
	"bufio"
	"io"
	"os"
	"sync"

	"github.com/goccy/go-json"
)

// Stdio 处理基于 stdin/stdout 的 JSON-RPC
type Stdio struct {
	reader *bufio.Scanner
	mu     sync.Mutex
}

func NewStdio() *Stdio {
	// 增加 Buffer 大小以支持大消息，最大支持 4MB
	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 4*1024*1024)

	return &Stdio{
		reader: scanner,
	}
}

func (s *Stdio) Read() (*JSONRPCMessage, error) {
	if !s.reader.Scan() {
		if err := s.reader.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	var msg JSONRPCMessage
	if err := json.Unmarshal(s.reader.Bytes(), &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *Stdio) Write(msg *JSONRPCMessage) error {
	p, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// 手动写入 stdout 并追加换行
	if _, err = os.Stdout.Write(p); err != nil {
		return err
	}
	_, err = os.Stdout.Write([]byte("\n"))
	return err
}

func (s *Stdio) Close() error {
	return nil
}
