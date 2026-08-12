package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

type CLIHeader struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
}

type CliServer struct {
	engine *Engine
	outMu  sync.Mutex // 保护 Stdout 的并发写入
}

func NewCliServer(engine *Engine) *CliServer {
	return &CliServer{engine: engine}
}

func (s *CliServer) Start() error {
	// 强制将标准日志重定向至 Stderr，防止日志污染 Stdout 的 JSON-RPC 报文
	log.SetOutput(os.Stderr)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 异步监听 Context 取消信号，主动关闭 Stdin 以解除 ReadBytes 的阻塞
	go func() {
		<-ctx.Done()
		_ = os.Stdin.Close()
	}()

	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, os.ErrClosed) || err == io.EOF {
				return nil
			}
			return fmt.Errorf("read error: %w", err)
		}

		raw := strings.TrimSpace(string(line))
		if raw == "" {
			continue
		}

		s.handleMessage(ctx, []byte(raw))
	}
}

func (s *CliServer) handleMessage(ctx context.Context, raw []byte) {
	// 语法解析错误 (-32700)
	if !json.Valid(raw) {
		s.writeBytes(BuildRPCError(nil, ErrCodeParseError, "Parse error: invalid JSON"))
		return
	}

	// RPC 结构校验 (-32600)
	var header RPCHeader
	_ = json.Unmarshal(raw, &header)

	if header.JSONRPC != "2.0" {
		if !header.IsNotification() {
			s.writeBytes(BuildRPCError(header.ParseID(), ErrCodeInvalidRequest, "Invalid Request: jsonrpc version must be '2.0'"))
		}
		return
	}

	isNotification := header.IsNotification()
	reqID := header.ParseID()

	// Panic 隔离防护
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MCP CLI] Recovered panic: %v", r)
			if !isNotification {
				s.writeBytes(BuildRPCError(reqID, ErrCodeInternalError, fmt.Sprintf("Internal error: panic %v", r)))
			}
		}
	}()

	// 传递请求给 Engine 执行
	resp, err := s.engine.ProcessMessage(ctx, raw)
	if err != nil {
		if !isNotification {
			s.writeBytes(BuildRPCError(reqID, ErrCodeInternalError, err.Error()))
		} else {
			log.Printf("[MCP CLI] Error processing notification: %v", err)
		}
		return
	}

	// Notification 或 Engine返回空 无需输出
	if isNotification || len(resp) == 0 {
		return
	}

	s.writeBytes(resp)
}

func (s *CliServer) writeBytes(data []byte) {
	s.outMu.Lock()
	defer s.outMu.Unlock()

	_, _ = os.Stdout.Write(data)
	_, _ = os.Stdout.Write([]byte("\n"))
}
