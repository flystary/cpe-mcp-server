package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type SseServer struct {
	addr   string
	engine *Engine

	mu       sync.RWMutex
	sessions map[string]chan []byte
}

func NewSseServer(addr string, engine *Engine) *SseServer {
	return &SseServer{
		addr:     addr,
		engine:   engine,
		sessions: make(map[string]chan []byte),
	}
}

func (s *SseServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", s.handleSSE)
	mux.HandleFunc("/message", s.handleMessage)
	return http.ListenAndServe(s.addr, mux)
}

// 生成安全唯一的 Session ID
func generateSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *SseServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 处理 CORS Header
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 为每个连接生成唯一的 Session ID
	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		sessionID = generateSessionID()
	}

	ch := make(chan []byte, 64)

	s.mu.Lock()
	// 如果已存在同名 session，先清理旧通道
	if oldCh, exists := s.sessions[sessionID]; exists {
		close(oldCh)
	}
	s.sessions[sessionID] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		if currentCh, ok := s.sessions[sessionID]; ok && currentCh == ch {
			delete(s.sessions, sessionID)
			close(ch)
		}
		s.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// 按照 MCP 规范，返回带 Session ID 的 endpoint
	fmt.Fprintf(w, "event: endpoint\ndata: /message?id=%s\n\n", sessionID)
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()

		case <-ticker.C:
			// 心跳包
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			return
		}
	}
}

func (s *SseServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	// CORS 预检处理
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		http.Error(w, "Missing session id", http.StatusBadRequest)
		return
	}

	// 限制 Payload 大小 (例如 4MB)，防止内存爆破
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Request body too large or unreadable", http.StatusBadRequest)
		return
	}

	// 提前检查 Session 是否存在
	s.mu.RLock()
	targetChan, active := s.sessions[sessionID]
	s.mu.RUnlock()

	if !active {
		http.Error(w, "Session expired or inactive", http.StatusBadRequest)
		return
	}

	// JSON 语法解析校验 (-32700)
	if !json.Valid(payload) {
		s.sendToSession(targetChan, BuildRPCError(nil, ErrCodeParseError, "Parse error: invalid JSON"))
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// RPC 报文头提取校验 (-32600)
	var header RPCHeader
	_ = json.Unmarshal(payload, &header)

	if header.JSONRPC != "2.0" {
		if !header.IsNotification() {
			s.sendToSession(targetChan, BuildRPCError(header.ParseID(), ErrCodeInvalidRequest, "Invalid Request: jsonrpc version must be '2.0'"))
		}
		w.WriteHeader(http.StatusAccepted)
		return
	}

	isNotification := header.IsNotification()
	reqID := header.ParseID()

	w.WriteHeader(http.StatusAccepted)

	// 脱离 r.Context() 的生命周期限制，使用独立 Background Context 并带超时
	taskCtx, taskCancel := context.WithTimeout(context.Background(), 10*time.Second)

	// 异步由 Engine 执行处理
	go func(ch chan []byte, ctx context.Context, cancel context.CancelFunc) {
		defer cancel()
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[MCP SSE] Panic in ProcessMessage: %v", rec)
				if !isNotification {
					s.sendToSession(ch, BuildRPCError(reqID, ErrCodeInternalError, fmt.Sprintf("Internal error: panic %v", rec)))
				}
			}
		}()

		resp, err := s.engine.ProcessMessage(ctx, payload)
		if err != nil {
			if !isNotification {
				s.sendToSession(ch, BuildRPCError(reqID, ErrCodeInternalError, err.Error()))
			}
			return
		}

		// 如果不是 Notification 且有响应，写入 SSE 通道
		if !isNotification && len(resp) > 0 {
			s.sendToSession(ch, resp)
		}
	}(targetChan, taskCtx, taskCancel)
}

func (s *SseServer) sendToSession(ch chan []byte, data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	select {
	case ch <- data:
	default:
		log.Printf("[MCP SSE] Session channel full, message dropped")
	}
}
